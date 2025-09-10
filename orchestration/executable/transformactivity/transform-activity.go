package transformactivity

import (
	"fmt"
	"net/http"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/jq"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type TransformActivity struct {
	executable.Activity
	definition config.TransformActivityDefinition
}

func NewTransformActivity(item config.Configurable, refs config.DataReferences) (*TransformActivity, error) {
	const semLogContext = "transform-activity::new"
	var err error

	ma := &TransformActivity{}
	ma.Cfg = item
	ma.Refs = refs

	maCfg := item.(*config.TransformActivity)
	ma.definition, err = config.UnmarshalTransformActivityDefinition(maCfg.Definition, refs)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return ma, nil
}

func (a *TransformActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.TransformActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.TransformActivityType)).Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.TransformActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.TransformActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	err = tcfg.WfCaseDeadlineExceeded(wfc.RequestTiming, wfc.RequestDeadline)
	if err != nil {
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	activityBegin := time.Now()
	defer func(begin time.Time) {
		wfc.RequestTiming += time.Since(begin)
		log.Info().Str(constants.SemLogActivity, a.Name()).Float64("wfc-timing.s", wfc.RequestTiming.Seconds()).Float64("deadline.s", wfc.RequestDeadline.Seconds()).Msg(semLogContext + " - wfc timing")
	}(activityBegin)

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
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	_ = wfc.SetHarEntryRequest(a.Name(), req, tcfg.PII)

	harResponse, err := a.Invoke(wfc, expressionCtx)
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
	}

	if harResponse != nil {
		_ = wfc.SetHarEntryResponse(a.Name(), harResponse, tcfg.PII)
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(harResponse.Status)
	}

	remappedStatusCode, err := a.ProcessResponseActionByStatusCode(harResponse.Status, a.Name(), a.Name(), wfc, nil, wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}, a.definition.OnResponseActions, true)
	if remappedStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
	}
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithError(err), smperror.WithErrorAmbit(a.Name()))
	}

	/*	actNdx := a.findResponseAction(harResponse.Status)
		if actNdx >= 0 {
			remappedStatusCode, err := a.processResponseAction(wfc, a.Name(), actNdx, harResponse)
			if remappedStatusCode != 0 {
				metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
			}
			if err != nil {
				wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
				a.SetMetrics(beginOf, metricsLabels)
				return err
			}
		}
	*/

	// _ = a.SetMetrics(beginOf, metricsLabels)
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)

	return err
}

func (a *TransformActivity) executeKazaamTransformation(kazaamId string, data []byte) ([]byte, error) {
	return kz.GetRegistry().Transform(kazaamId, data)
}

func (a *TransformActivity) executeJQTransformation(jqId string, data []byte) ([]byte, error) {
	return jq.GetRegistry().Transform(jqId, data)
}

func (a *TransformActivity) resolveAndExecuteKazaamTransformation(wfc *wfcase.WfCase, xForm *xforms.TransformReference, resolver *wfexpressions.Evaluator) ([]byte, error) {
	const semLogContext = "transform-activity::resolve-and-execute-kazaam-transformation"

	resolvedTransformation, err := resolver.EvaluateTemplate(string(xForm.Data), wfc.TemplateFunctions())
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	data, err := resolver.BodyAsByteArray()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return kz.ApplyKazaamTransformation(resolvedTransformation, data)
}

func (a *TransformActivity) executeJsonExt2JsonTransformation(data []byte) ([]byte, error) {
	b, err := util.JsonExtended2JsonConv(data)
	return b, err
}

func (a *TransformActivity) executeMergeTransformation(wfc *wfcase.WfCase, mergeXForm []byte, currentData []byte) ([]byte, error) {
	const semLogContext = "transform-activity::execute-merge-transformation"

	xform, err := NewTransformActivityMergeXForm(mergeXForm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return xform.Execute(wfc, currentData)
}

func (a *TransformActivity) executeTemplateTransformation(wfc *wfcase.WfCase, bodyTemplate []byte, resolver *wfexpressions.Evaluator) ([]byte, error) {

	s, _, err := varResolver.ResolveVariables(string(bodyTemplate), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}

	b, err := wfc.ProcessTemplate(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (a *TransformActivity) Invoke(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference) (*har.Response, error) {

	const semLogContext = "transform-activity::invoke"
	var err error
	resolver, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		r := har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "text/plain", []byte(err.Error()), nil)
		return r, err
	}
	// I'm setting the name since it gets modified and is not any more valid.
	resolver.Name = fmt.Sprintf("%s-%s", a.Name(), "req")

	var b []byte
	for _, xform := range a.definition.Transforms {
		switch xform.Typ {
		case config.XFormTemplate:
			b, err = a.executeTemplateTransformation(wfc, xform.Data, resolver)
		case config.XFormKazaamDynamic:
			b, err = a.resolveAndExecuteKazaamTransformation(wfc, &xform, resolver)
		case config.XFormKazaam:
			b, err = resolver.BodyAsByteArray()
			b, err = a.executeKazaamTransformation(xform.Id, b)
		case config.XFormJQ:
			b, err = resolver.BodyAsByteArray()
			b, err = a.executeJQTransformation(xform.Id, b)
		case config.XFormMerge:
			b, err = resolver.BodyAsByteArray()
			b, err = a.executeMergeTransformation(wfc, xform.Data, b)
		case config.XFormJsonExt2Json:
			b, err = resolver.BodyAsByteArray()
			b, err = a.executeJsonExt2JsonTransformation(b)
		default:
			log.Warn().Str("type", xform.Typ).Msg(semLogContext + " unsupported transformation")
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			r := har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "text/plain", []byte(err.Error()), nil)
			return r, err
		}

		err = resolver.WithBody(constants.ContentTypeApplicationJson, b, "")
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			r := har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "text/plain", []byte(err.Error()), nil)
			return r, err
		}
	}

	var r *har.Response
	r = &har.Response{
		Status:      http.StatusOK,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(http.StatusOK),
		HeadersSize: -1,
		BodySize:    int64(len(b)),
		Headers:     []har.NameValuePair{},
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeApplicationJson,
			Size:     int64(len(b)),
			Data:     b,
		},
	}

	return r, nil
}

func (a *TransformActivity) newRequestDefinition(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference) (*har.Request, error) {

	const semLogContext = "transform-activity::new-request-definition"

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(0000)
	ub.WithScheme("activity")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s", string(config.TransformActivityType), a.Name()))
	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))

	b, _, err := wfc.GetBodyInHarEntry(expressionCtx, true)
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

func (a *TransformActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}

func (a *TransformActivity) processResponseAction(wfc *wfcase.WfCase, activityName string, actionIndex int, resp *har.Response) (int, error) /* *smperror.SymphonyError */ {
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

func chooseTransformation(wfc *wfcase.WfCase, trs []xforms.TransformReference) (string, error) {
	for _, t := range trs {

		b := true
		if t.Guard != "" {
			b = wfc.EvalBoolExpression(t.Guard)
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

		if wfc.EvalBoolExpression(e.Guard) {
			return i
		}
	}

	return -1
}

func (a *TransformActivity) findResponseAction(statusCode int) int {

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
