package endpointactivity

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cacheoperation"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/linkedservices"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"net/http"
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

func (ep Endpoint) FullId(activityName string) string {
	return EndpointFullId(activityName, ep.Id)
}

func EndpointFullId(activityName, endpointId string) string {
	return fmt.Sprintf("%s@%s", activityName, endpointId)
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

func registerTransformations(ts []kzxform.TransformReference, refs config.DataReferences) error {
	tReg := kzxform.GetRegistry()
	if tReg == nil {
		err := errors.New("transformation registry not initialized")
		return err
	}

	for _, tref := range ts {
		trasDef, _ := refs.Find(tref.DefinitionRef)
		if len(trasDef) == 0 {
			return fmt.Errorf("cannot find transformation %s definition from %s", tref.Id, tref.DefinitionRef)
		}

		tref.Data = trasDef
		err := tReg.AddTransformation(tref)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *EndpointActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.EndpointActivityType) + "::execute"

	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.EndpointActivityType)).Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	cfg, ok := a.Cfg.(*config.EndpointActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.EndpointActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	/*
		expressionCtx, err := wfc.ResolveExpressionContextName(a.Cfg.ExpressionContextNameStringReference())
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
			return err
		}
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext + " start")
	*/

	if len(cfg.ProcessVars) > 0 {
		// note the ignoreNonApplicationJsonResponseContent has been set to false since it doesn't apply to the request processing
		expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext)
		err = wfc.SetVars(expressionCtx, cfg.ProcessVars, "", false)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	for _, ep := range a.Endpoints {

		beginOf := time.Now()
		metricsLabels := a.MetricsLabels(ep)

		resolver, err := a.GetEvaluator(wfc)
		if err != nil {
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		var harResponse *har.Response
		var cacheCfg config.CacheConfig
		var cacheEnabled bool
		cacheEnabled, err = ep.Definition.CacheConfig.Enabled()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}

		if cacheEnabled {
			cacheCfg, err = a.resolveCacheConfig(wfc, resolver, ep.Definition.CacheConfig, a.Refs)
			if err != nil {
				// The get of the cache triggers an error only.
				log.Error().Err(err).Msg(semLogContext)
			} else {
				harResponse, err = a.resolveResponseFromCache(wfc, ep.FullId(a.Name()), cacheCfg.Key /* ep.Definition.Path */, cacheCfg)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					if harResponse == nil {
						return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
					}
				}
			}
		}

		if harResponse == nil || harResponse.Status != http.StatusOK {
			req, err := a.newRequestDefinition(wfc, ep)
			if err != nil {
				wfc.AddBreadcrumb(ep.FullId(a.Name()), ep.Description, err)
				metricsLabels[MetricIdStatusCode] = "500"
				_ = a.SetMetrics(beginOf, metricsLabels)
				return smperror.NewExecutableServerError(smperror.WithErrorAmbit(ep.Name), smperror.WithStep(ep.Id), smperror.WithCode("HTTP"), smperror.WithErrorMessage(err.Error()))
			}

			_ = wfc.SetHarEntryRequest(ep.FullId(a.Name()), req, ep.PII)

			entry, err := a.Invoke(wfc, ep, req)
			if entry != nil {
				harResponse = entry.Response
			}
			_ = wfc.SetHarEntryResponse(ep.FullId(a.Name()), harResponse, ep.PII)
			metricsLabels[MetricIdHttpStatusCode] = fmt.Sprint(harResponse.Status)
			metricsLabels[MetricIdStatusCode] = fmt.Sprint(harResponse.Status)

			if cacheEnabled && harResponse.Status == http.StatusOK {
				err = a.saveResponseToCache(cacheCfg, harResponse.Content.Data)
				// err = cacheoperation.Set(cacheCfg.LinkedServiceRef, cacheCfg.Key, harResponse.Content.Data, cachelks.WithNamespace(cacheCfg.Namespace))
				if err != nil {
					// The set of the cache triggers an error only.
					log.Error().Err(err).Msg(semLogContext)
				}
			}
		}

		remappedStatusCode, err := a.ProcessResponseActionByStatusCode(
			harResponse.Status, a.Name(), util.StringCoalesce(ep.Id, ep.Name), wfc, nil, wfcase.HarEntryReference{Name: ep.FullId(a.Name()), UseResponse: true}, ep.Definition.OnResponseActions, ep.Definition.IgnoreNonApplicationJsonResponseContent)
		if remappedStatusCode > 0 {
			metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
		}
		if err != nil {
			wfc.AddBreadcrumb(ep.Id, ep.Description, err)
			_ = a.SetMetrics(beginOf, metricsLabels)
			return err
		}

		/*
			actNdx := findResponseAction(ep, harResponse.Status)
			if actNdx >= 0 {
				remappedStatusCode, err := processResponseAction(wfc, a.Name(), ep, actNdx, harResponse)
				if remappedStatusCode != 0 {
					metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
				}
				if err != nil {
					wfc.AddBreadcrumb(ep.Id, ep.Description, err)
					_ = a.SetMetrics(beginOf, metricsLabels)
					return err
				}
			}
		*/
		_ = a.SetMetrics(beginOf, metricsLabels)
		wfc.AddBreadcrumb(ep.Id, ep.Description, nil)
	}

	return nil
}

/*
func (a *EndpointActivity) getResolver(wfc *wfcase.WfCase) (*wfexpressions.Evaluator, error) {
	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		return nil, err
	}

	resolver, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", false)
	if err != nil {
		return nil, err
	}

	return resolver, nil
}
*/

/*
	func processResponseAction(wfc *wfcase.WfCase, activityName string, ep Endpoint, actionIndex int, resp *har.Response) (int, error) {
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

		resolverContextReference := wfcase.ResolverContextReference{Name: ep.FullId(activityName), UseResponse: true}

		if len(act.ProcessVars) > 0 {
			err := wfc.SetVars(resolverContextReference, act.ProcessVars, transformId, ignoreNonJSONResponseContent)
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

			m, err := wfc.ResolveStrings(resolverContextReference, []string{e.Code, e.Message, e.Description, step}, "", ignoreNonJSONResponseContent)
			if err != nil {
				log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
				return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
			}
			return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]))
		}

		return 0, nil
	}
*/

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

	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		return nil, err
	}

	// note the ignoreNonApplicationJsonResponseContent has been set to false since it doesn't apply to the request processing
	resolver, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", false)
	if err != nil {
		return nil, err
	}

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(ep.Definition.PortAsInt())
	ub.WithScheme(ep.Definition.Scheme)

	s, _, err := varResolver.ResolveVariables(ep.Definition.HostName, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}
	ub.WithHostname(s)

	s, _, err = varResolver.ResolveVariables(ep.Definition.Path, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}
	ub.WithPath(s)

	opts = append(opts, har.WithMethod(ep.Definition.Method))
	opts = append(opts, har.WithUrl(ub.Url()))

	for _, h := range ep.Definition.Headers {
		r, _, err := varResolver.ResolveVariables(h.Value, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}
		opts = append(opts, har.WithHeader(har.NameValuePair{Name: h.Name, Value: r}))
	}

	for _, qs := range ep.Definition.QueryString {
		if wfc.EvalBoolExpression(qs.Guard) {
			r, _, err := varResolver.ResolveVariables(qs.Value, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
			if err != nil {
				return nil, err
			}
			opts = append(opts, har.WithQueryParam(har.NameValuePair{Name: qs.Name, Value: r}))
		}
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
		Headers:     []har.NameValuePair{},
		BodySize:    -1,
	}
	for _, o := range opts {
		o(&req)
	}

	return &req, nil
}

func (a *EndpointActivity) newRequestDefinitionBody(wfc *wfcase.WfCase, ep Endpoint, resolver *wfexpressions.Evaluator) (har.RequestOption, error) {

	var bodyContent []byte
	if ep.Definition.Body.ExternalValue != "" {
		bodyContent, _ = a.Refs.Find(ep.Definition.Body.ExternalValue)
	} else {
		bodyContent = []byte(ep.Definition.Body.Value)
	}
	s, _, err := varResolver.ResolveVariables(string(bodyContent), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
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

func (a *EndpointActivity) resolveCacheConfig(wfc *wfcase.WfCase, resolver *wfexpressions.Evaluator, cacheConfig config.CacheConfig, refs config.DataReferences) (config.CacheConfig, error) {
	cfg := cacheConfig
	if refs.IsPresent(cacheConfig.Key) {
		if key, ok := refs.Find(cacheConfig.Key); ok {
			cfg.Key = string(key)
		}
	}

	s, _, err := varResolver.ResolveVariables(cfg.Key, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return cfg, err
	}

	b1, err := wfc.ProcessTemplate(s)
	if err != nil {
		return cfg, err
	}

	cfg.Key = string(b1)
	return cfg, err
}

func (a *EndpointActivity) resolveResponseFromCache(wfc *wfcase.WfCase, endpointId, endpointPath string, cacheConfig config.CacheConfig) (*har.Response, error) {
	cacheHarEntry, err := cacheoperation.Get(
		cacheConfig.LinkedServiceRef,
		a.Name()+";cache=true",
		cacheConfig.Key,
		constants.ContentTypeApplicationJson,
		cachelks.WithNamespace(cacheConfig.Namespace), cachelks.WithHarPath(fmt.Sprintf("%s;cache=true", endpointPath)))
	if err != nil {
		return nil, err
	}

	// the id takes the activity name in case ok because no other entry will be present. In case of cache miss ad additional entry will be there
	// together with the un-cached invokation
	entryId := endpointId
	if cacheHarEntry.Response.Status != http.StatusOK {
		entryId = a.Name() + ";cache=true"
	}

	_ = wfc.SetHarEntry(entryId, cacheHarEntry)
	return cacheHarEntry.Response, nil
}

func (a *EndpointActivity) saveResponseToCache(cacheConfig config.CacheConfig, data []byte) error {
	err := cacheoperation.Set(cacheConfig.LinkedServiceRef, cacheConfig.Key, data, cachelks.WithNamespace(cacheConfig.Namespace), cachelks.WithTTTL(cacheConfig.Ttl))
	if err != nil {
		return err
	}

	return nil
}
