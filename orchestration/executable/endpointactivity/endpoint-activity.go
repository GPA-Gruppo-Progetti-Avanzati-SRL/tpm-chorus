package endpointactivity

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/linkedservices"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"strings"
	"time"
)

const (
	MetricIdActivityType           = "type"
	MetricIdActivityName           = "name"
	MetricIdEndpointDefinitionPath = "endpoint"
	MetricIdEndpointId             = "endpoint-id"
	MetricIdEndpointName           = "endpoint-name"
	MetricIdStatusCode             = "status-code"
	MetricIdMethod                 = "http-method"
	MetricIdHttpStatusCode         = "http-status-code"
)

type Endpoint struct {
	Id          string
	Name        string
	Description string
	Definition  *config.EndpointDefinition
	PII         config.PersonallyIdentifiableInformation
}

type EndpointActivity struct {
	executable.Activity
	Endpoints []Endpoint
}

func NewEndpointActivity(item config.Configurable, refs config.DataReferences) (*EndpointActivity, error) {

	ea := &EndpointActivity{}
	ea.Cfg = item
	ea.Refs = refs

	tcfg := item.(*config.EndpointActivity)
	for _, epcfg := range tcfg.Endpoints {

		epCfgDef, _ := refs.Find(epcfg.Definition)
		if len(epCfgDef) == 0 {
			return nil, fmt.Errorf("cannot find endpoint (%s:%s) definition from %s", epcfg.Id, epcfg.Name, epcfg.Definition)
		}

		epDef := config.EndpointDefinition{}
		err := yaml.Unmarshal(epCfgDef, &epDef)
		if err != nil {
			return nil, err
		}

		if epDef.Body.ExternalValue != "" && !refs.IsPresent(epDef.Body.ExternalValue) {
			return nil, fmt.Errorf("cannot find endpoint (%s:%s) body reference from %s", epcfg.Id, epcfg.Name, epDef.Body.ExternalValue)
		}

		for _, onRespAct := range epDef.OnResponseActions {
			err = registerTransformations(onRespAct.Transforms, refs)
			if err != nil {
				return nil, err
			}
		}

		ep := Endpoint{Id: epcfg.Id, Name: epcfg.Name, Description: epcfg.Description, Definition: &epDef, PII: epcfg.PII}
		ea.Endpoints = append(ea.Endpoints, ep)
	}

	return ea, nil
}

func registerTransformations(ts []config.TransformReference, refs config.DataReferences) error {
	tReg := transform.GetRegistry()
	for _, tref := range ts {
		trasDef, _ := refs.Find(tref.DefinitionRef)
		if len(trasDef) == 0 {
			return fmt.Errorf("cannot find transformation %s definition from %s", tref.Id, tref.DefinitionRef)
		}

		trsf := transform.Config{}
		err := yaml.Unmarshal(trasDef, &trsf)
		if err != nil {
			return err
		}

		err = tReg.Add(trsf)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *EndpointActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = "rest-activity::execute"

	var err error

	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "echo").Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "endpoint").Msg(semLogContext + " start")

	cfg, ok := a.Cfg.(*config.EndpointActivity)
	if !ok {
		err := fmt.Errorf("this is weird %v is not (*config.EndpointActivity)", a.Cfg)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Msgf(err.Error())
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	// if len(cfg.ProcessVars) > 0 {
	// note the ignoreNonApplicationJsonResponseContent has been set to false since it doesn't apply to the request processing
	err = wfc.SetVars("request", cfg.ProcessVars, "", false)
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}
	//}

	for _, ep := range a.Endpoints {

		beginOf := time.Now()
		metricsLabels := a.MetricsLabels(ep)

		req, err := a.newRequestDefinition(wfc, ep)
		if err != nil {
			wfc.AddBreadcrumb(ep.Id, ep.Description, err)
			metricsLabels[MetricIdStatusCode] = "500"
			a.SetMetrics(beginOf, metricsLabels)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(ep.Name), smperror.WithStep(ep.Id), smperror.WithCode("HTTP"), smperror.WithErrorMessage(err.Error()))
		}

		_ = wfc.AddEndpointRequestData(ep.Id, req, ep.PII)

		entry, err := a.Invoke(wfc, ep, req)
		var resp *har.Response
		if entry != nil {
			resp = entry.Response
		}
		_ = wfc.AddEndpointResponseData(ep.Id, resp, ep.PII)

		metricsLabels[MetricIdHttpStatusCode] = fmt.Sprint(resp.Status)
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(resp.Status)
		actNdx := findResponseAction(ep, resp.Status)
		if actNdx >= 0 {
			remappedStatusCode, err := processResponseAction(wfc, a.Name(), ep, actNdx, resp)
			if remappedStatusCode != 0 {
				metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
			}
			if err != nil {
				wfc.AddBreadcrumb(ep.Id, ep.Description, err)
				a.SetMetrics(beginOf, metricsLabels)
				return err
			}
		}

		a.SetMetrics(beginOf, metricsLabels)
		wfc.AddBreadcrumb(ep.Id, ep.Description, nil)
	}

	return nil
}

func processResponseAction(wfc *wfcase.WfCase, activityName string, ep Endpoint, actionIndex int, resp *har.Response) (int, error) /* *smperror.SymphonyError */ {
	act := ep.Definition.OnResponseActions[actionIndex]

	ignoreNonJSONResponseContent := ep.Definition.IgnoreNonApplicationJsonResponseContent
	if !ignoreNonJSONResponseContent {
		ignoreNonJSONResponseContent = act.IgnoreNonApplicationJsonResponseContent
	}

	transformId, err := chooseTransformation(wfc, act.Transforms)
	if err != nil {
		log.Error().Err(err).Str("ctx", ep.Id).Str("request-id", wfc.GetRequestId()).Msg("processResponseAction: error in selecting transformation")
		return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(ep.Name), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
	}

	if len(act.ProcessVars) > 0 {
		err := wfc.SetVars(ep.Id, act.ProcessVars, transformId, ignoreNonJSONResponseContent)
		if err != nil {
			log.Error().Err(err).Str("ctx", ep.Id).Str("request-id", wfc.GetRequestId()).Msg("processResponseAction: error in setting variables")
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(ep.Name), smperror.WithCode("500"), smperror.WithErrorMessage("error processing response body"), smperror.WithDescription(err.Error()))
		}
	}

	if ndx := chooseError(wfc, act.Errors); ndx >= 0 {

		e := act.Errors[ndx]
		ambit := e.Ambit
		if ambit == "" {
			ambit = activityName
		}

		step := e.Step
		if step == "" {
			step = ep.Name
		}
		if step == "" {
			step = ep.Id
		}

		statusCode := int(resp.Status)
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}

		m, err := wfc.ResolveStrings(ep.Id, []string{e.Code, e.Message, e.Description, step}, "", ignoreNonJSONResponseContent)
		if err != nil {
			log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
		}
		return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]))
	}

	return 0, nil
}

func chooseTransformation(wfc *wfcase.WfCase, trs []config.TransformReference) (string, error) {
	for _, t := range trs {

		b := true
		if t.Guard != "" {
			b = wfc.EvalExpression(t.Guard)
		}

		if b {
			return t.Id, nil
		}
	}

	return "", nil
}

func chooseError(wfc *wfcase.WfCase, errors []config.ErrorInfo) int {
	for i, e := range errors {
		if e.Guard == "" {
			return i
		}

		if wfc.EvalExpression(e.Guard) {
			return i
		}
	}

	return -1
}

func findResponseAction(ep Endpoint, statusCode int) int {

	matchedAction := -1
	defaultAction := -1
	for ndx, act := range ep.Definition.OnResponseActions {
		if act.StatusCode == statusCode {
			matchedAction = ndx
			break
		}

		if act.StatusCode == -1 {
			defaultAction = ndx
		}
	}

	if matchedAction < 0 && defaultAction >= 0 {
		matchedAction = defaultAction
	}

	return matchedAction
}

func (a *EndpointActivity) Invoke(wfc *wfcase.WfCase, ep Endpoint, req *har.Request) (*har.Entry, error) {

	const semLogContext = "endpoint-activity::invoke"
	const semLogContextOverrideOption = " using endpoint specific value"
	opts := []restclient.Option{restclient.WithSpan(wfc.Span)}
	if ep.Definition.HttpClientOptions != nil {
		if ep.Definition.HttpClientOptions.RestTimeout != 0 {
			log.Info().Dur("timeout", ep.Definition.HttpClientOptions.RestTimeout).Str("endpoint", ep.Id).Msg(semLogContext + semLogContextOverrideOption)
			opts = append(opts, restclient.WithTimeout(ep.Definition.HttpClientOptions.RestTimeout))
		}

		log.Info().Dur("retry-wait-time", ep.Definition.HttpClientOptions.RetryWaitTime).Str("endpoint", ep.Id).Msg(semLogContext + semLogContextOverrideOption)
		opts = append(opts, restclient.WithRetryWaitTime(ep.Definition.HttpClientOptions.RetryWaitTime))

		log.Info().Dur("retry-max-wait-time", ep.Definition.HttpClientOptions.RetryMaxWaitTime).Str("endpoint", ep.Id).Msg(semLogContext + semLogContextOverrideOption)
		opts = append(opts, restclient.WithRetryWaitTime(ep.Definition.HttpClientOptions.RetryMaxWaitTime))

		log.Info().Int("retry-count", ep.Definition.HttpClientOptions.RetryCount).Str("endpoint", ep.Id).Msg(semLogContext + semLogContextOverrideOption)
		opts = append(opts, restclient.WithRetryCount(ep.Definition.HttpClientOptions.RetryCount))

		log.Info().Interface("retry-on-errors", ep.Definition.HttpClientOptions.RetryOnHttpError).Str("endpoint", ep.Id).Msg(semLogContext + semLogContextOverrideOption)
		opts = append(opts, restclient.WithRetryOnHttpError(ep.Definition.HttpClientOptions.RetryOnHttpError))
	}

	cli, err := linkedservices.GetRestClientProvider(opts...)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.Execute(req, restclient.ExecutionWithOpName(ep.Id))
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return resp, err
	}
	log.Trace().Int("status-code", resp.Response.Status).Int("num-headers", len(resp.Response.Headers)).Int64("content-length", resp.Response.BodySize).Msg(semLogContext)

	/* the handling of the IgnoreNonApplicationJsonResponseContent has been moved down the chain in the processing of the response action */
	ct := resp.Response.Content.MimeType
	if err == nil && !strings.HasPrefix(ct, constants.ContentTypeApplicationJson) && resp.Response.Status != 200 && resp.Response.BodySize > 0 {
		// err = fmt.Errorf("%s", string(resp.Content.Data))
		log.Debug().Str("content-type", ct).Msg(semLogContext + " content is not the usual " + constants.ContentTypeApplicationJson)
	}

	return resp, err
}

func (a *EndpointActivity) newRequestDefinition(wfc *wfcase.WfCase, ep Endpoint) (*har.Request, error) {

	// note the ignoreNonApplicationJsonResponseContent has been set to false since it doesn't apply to the request processing
	resolver, err := wfc.GetResolverForEntry("request", true, "", false)
	if err != nil {
		return nil, err
	}

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(ep.Definition.PortAsInt())
	ub.WithScheme(ep.Definition.Scheme)

	s, _, err := varResolver.ResolveVariables(ep.Definition.HostName, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return nil, err
	}
	ub.WithHostname(s)

	s, _, err = varResolver.ResolveVariables(ep.Definition.Path, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return nil, err
	}
	ub.WithPath(s)

	opts = append(opts, har.WithMethod(ep.Definition.Method))
	opts = append(opts, har.WithUrl(ub.Url()))

	for _, h := range ep.Definition.Headers {
		r, _, err := varResolver.ResolveVariables(h.Value, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
		if err != nil {
			return nil, err
		}
		opts = append(opts, har.WithHeader(har.NameValuePair{Name: h.Name, Value: r}))
	}

	if !ep.Definition.Body.IsZero() {
		opt, err := a.newRequestDefinitionBody(wfc, ep, resolver)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}

	req := har.Request{
		HTTPVersion: "1.1",
		Cookies:     []har.Cookie{},
		QueryString: []har.NameValuePair{},
		HeadersSize: -1,
		BodySize:    -1,
	}
	for _, o := range opts {
		o(&req)
	}

	return &req, nil
}

func (a *EndpointActivity) newRequestDefinitionBody(wfc *wfcase.WfCase, ep Endpoint, resolver *wfcase.ProcessVarResolver) (har.RequestOption, error) {

	bodyContent, _ := a.Refs.Find(ep.Definition.Body.ExternalValue)
	s, _, err := varResolver.ResolveVariables(string(bodyContent), varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return nil, err
	}

	if ep.Definition.Body.Type == "simple" {
		return har.WithBody([]byte(s)), nil
	}

	b, err := wfc.ProcessTemplate(s)
	if err != nil {
		return nil, err
	}

	return har.WithBody(b), nil

}

func (a *EndpointActivity) MetricsLabels(ep Endpoint) prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType:           string(a.Cfg.Type()),
		MetricIdActivityName:           a.Name(),
		MetricIdEndpointDefinitionPath: ep.Definition.Path,
		MetricIdEndpointId:             ep.Id,
		MetricIdEndpointName:           ep.Name,
		MetricIdStatusCode:             "-1",
		MetricIdHttpStatusCode:         "-1",
		MetricIdMethod:                 ep.Definition.Method,
	}

	return metricsLabels
}
