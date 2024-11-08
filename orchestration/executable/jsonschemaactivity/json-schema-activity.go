package jsonschemaactivity

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/jsonschemaregistry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type JsonSchemaActivity struct {
	executable.Activity
	definition config.JsonSchemaActivityDefinition
}

func NewJsonSchemaActivity(item config.Configurable, refs config.DataReferences) (*JsonSchemaActivity, error) {
	const semLogContext = "json-schema-activity::new"
	var err error

	ma := &JsonSchemaActivity{}
	ma.Cfg = item
	ma.Refs = refs

	maCfg := item.(*config.JsonSchemaActivity)
	ma.definition, err = config.UnmarshalJsonSchemaActivityDefinition(maCfg.Name(), maCfg.Definition, refs)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return ma, nil
}

func (a *JsonSchemaActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.JsonSchemaActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.JsonSchemaActivityType)).Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.JsonSchemaActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.JsonSchemaActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}
	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

	if len(tcfg.ProcessVars) > 0 {
		err := wfc.SetVars(expressionCtx, tcfg.ProcessVars, "", false)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func() { a.SetMetrics(beginOf, metricsLabels) }()

	req, err := a.newRequestDefinition(wfc, expressionCtx) // TODO calcolare lo statement
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		metricsLabels[MetricIdStatusCode] = "500"
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithCode("MONGO"), smperror.WithErrorMessage(err.Error()))
	}

	_ = wfc.SetHarEntryRequest(a.Name(), req, config.PersonallyIdentifiableInformation{})

	harResponse, err := a.Invoke(wfc, expressionCtx)
	if harResponse != nil {
		_ = wfc.SetHarEntryResponse(a.Name(), harResponse, config.PersonallyIdentifiableInformation{})
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(harResponse.Status)
	}

	remappedStatusCode, err := a.ProcessResponseActionByStatusCode(harResponse.Status, a.Name(), a.Name(), wfc, nil, wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}, a.definition.OnResponseActions, true)
	if remappedStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
	}
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		_ = a.SetMetrics(beginOf, metricsLabels)
		return err
	}

	// _ = a.SetMetrics(beginOf, metricsLabels)
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)

	return err
}

func (a *JsonSchemaActivity) Invoke(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference) (*har.Response, error) {

	const semLogContext = "json-schema-activity::invoke"
	var err error
	data, err := wfc.GetBodyInHarEntry(expressionCtx, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		r := har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "text/plain", []byte(err.Error()), nil)
		return r, err
	}

	err = jsonschemaregistry.Validate(a.Name(), a.definition.SchemaRef, data)
	if err != nil {
		log.Info().Err(err).Msg(semLogContext)
		r := har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "text/plain", []byte(err.Error()), nil)
		return r, err
	}

	var r *har.Response
	r = &har.Response{
		Status:      http.StatusOK,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(http.StatusOK),
		HeadersSize: -1,
		BodySize:    int64(len(http.StatusText(http.StatusOK))),
		Headers:     []har.NameValuePair{},
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeTextPlain,
			Size:     int64(len(http.StatusText(http.StatusOK))),
			Data:     []byte(http.StatusText(http.StatusOK)),
		},
	}

	return r, nil
}

func (a *JsonSchemaActivity) newRequestDefinition(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference) (*har.Request, error) {

	const semLogContext = "json-schema-activity::new-request-definition"

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(0000)
	ub.WithScheme("activity")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s", string(config.JsonSchemaActivityType), a.Name()))
	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))

	b, err := wfc.GetBodyInHarEntry(expressionCtx, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}
	opts = append(opts, har.WithBody(b))

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

const (
	MetricIdActivityType = "type"
	MetricIdActivityName = "name"
	MetricIdOpType       = "op-type"
	MetricIdStatusCode   = "status-code"
)

func (a *JsonSchemaActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}

/*
func (a *JsonSchemaActivity) processResponseAction(wfc *wfcase.WfCase, activityName string, actionIndex int, resp *har.Response) (int, error)  {
	act := a.definition.OnResponseActions[actionIndex]

	transformId, err := chooseTransformation(wfc, act.Transforms)
	if err != nil {
		log.Error().Err(err).Str("request-id", wfc.GetRequestId()).Msg("processResponseAction: error in selecting transformation")
		return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(a.Name()), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
	}

	contextReference := wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}

	if len(act.ProcessVars) > 0 {
		err := wfc.SetVars(contextReference, act.ProcessVars, transformId, false)
		if err != nil {
			log.Error().Err(err).Str("ctx", a.Name()).Str("request-id", wfc.GetRequestId()).Msg("processResponseAction: error in setting variables")
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(a.Name()), smperror.WithCode("500"), smperror.WithErrorMessage("error processing response body"), smperror.WithDescription(err.Error()))
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
			step = a.Name()
		}
		if step == "" {
			step = a.Name()
		}

		statusCode := int(resp.Status)
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}

		m, err := wfc.ResolveStrings(contextReference, []string{e.Code, e.Message, e.Description, step}, "", false)
		if err != nil {
			log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
		}
		return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]))
	}

	return 0, nil
}

func chooseTransformation(wfc *wfcase.WfCase, trs []transform.TransformReference) (string, error) {
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

func (a *JsonSchemaActivity) findResponseAction(statusCode int) int {

	matchedAction := -1
	defaultAction := -1
	for ndx, act := range a.definition.OnResponseActions {
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
*/
