package orchestration

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/responseactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	kzxform2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	ChorusLoopActivityIteratorValueVarName = "_chorus_loop_iterator"
)

type LoopControlFlow struct {
	cfg          config.LoopControlFlowDefinition
	current      int
	start        int
	end          int
	step         int
	inputRequest *har.Request
}

func (cf *LoopControlFlow) HasNext(wfc *wfcase.WfCase) bool {
	if cf.cfg.BreakCondition != "" {
		if wfc.EvalBoolExpression(cf.cfg.BreakCondition) {
			return false
		}
	}

	if cf.step > 0 {
		return cf.current < cf.end
	} else {
		return cf.current >= 0
	}
}

func (cf *LoopControlFlow) Next(wfc *wfcase.WfCase, evaluator *wfexpressions.Evaluator) ([]byte, error) {
	const semLogContext = "loop-activity::loop-control-flow-next"

	_ = wfc.Vars.Set(ChorusLoopActivityIteratorValueVarName, cf.current, false, 0, false)

	var b []byte
	if cf.cfg.XForm.Id != "" {
		var err error
		switch cf.cfg.XForm.Typ {

		case config.XFormKazaamDynamic:
			b, err = cf.resolveAndExecuteKazaamTransformation(wfc, &cf.cfg.XForm, evaluator)
		case config.XFormKazaam:
			b, err = evaluator.BodyAsByteArray()
			b, err = cf.executeKazaamTransformation(cf.cfg.XForm.Id, b)
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
	}

	if cf.step > 0 {
		cf.current += cf.step

	} else {
		cf.current -= cf.step
	}

	return b, nil
}

func (a *LoopControlFlow) executeKazaamTransformation(kazaamId string, data []byte) ([]byte, error) {
	return kzxform2.GetRegistry().Transform(kazaamId, data)
}

func (a *LoopControlFlow) resolveAndExecuteKazaamTransformation(wfc *wfcase.WfCase, xForm *kzxform2.TransformReference, resolver *wfexpressions.Evaluator) ([]byte, error) {
	const semLogContext = "loop-activity::resolve-and-execute-kazaam-transformation"

	// Missing template functions.
	resolvedTransformation, err := resolver.EvaluateTemplate(string(xForm.Data), nil)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	data, err := resolver.BodyAsByteArray()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return kzxform2.ApplyKazaamTransformation(resolvedTransformation, data)
}

func InitLoopControlFlow(cfg config.LoopControlFlowDefinition, evaluator *wfexpressions.Evaluator) (*LoopControlFlow, error) {
	const semLogContext = string(config.LoopActivityType) + "::init-control-flow"
	var err error

	c := &LoopControlFlow{
		cfg:     cfg,
		current: 0,
	}

	var v interface{}
	switch cfg.Typ {
	case config.LoopControlFLowFor:
		v, err = evaluator.InterpolateAndEval(cfg.Start)
		if err == nil {
			c.start, err = convToInt(v)
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		v, err = evaluator.InterpolateAndEval(cfg.End)
		if err == nil {
			c.end, err = convToInt(v)
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		v, err = evaluator.InterpolateAndEval(cfg.Step)
		if err == nil {
			c.step, err = convToInt(v)
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

	default:
		err = errors.New("control flow type not recognized: " + cfg.Typ)
	}

	return c, err
}

func convToInt(val interface{}) (int, error) {
	const semLogContext = string(config.LoopActivityType) + "::conv-to-int"

	var res int
	var err error
	switch val := val.(type) {
	case float64:
		res = int(val)
	case int:
		res = int(val)
	default:
		err = errors.New("value should be numeric")
		log.Error().Err(err).Msg(semLogContext)
	}

	return res, err
}

type LoopActivity struct {
	executable.Activity
	definition        config.LoopActivityDefinition
	bodyOrchestration Orchestration
}

func NewLoopActivity(item config.Configurable, refs config.DataReferences, mapOfNestedOrcs map[string]Orchestration) (*LoopActivity, error) {
	var err error

	ea := &LoopActivity{}
	ea.Cfg = item
	ea.Refs = refs

	eaCfg, ok := item.(*config.LoopActivity)
	if !ok {
		err := fmt.Errorf("this is weird %T is not %s config type", item, config.LoopActivityType)
		return nil, err
	}

	ea.definition, err = config.UnmarshalLoopActivityDefinition(eaCfg.Definition, refs)
	if err != nil {
		return nil, err
	}

	if ea.definition.OrchestrationId == "" {
		err = errors.New("loop activity must specify id of orchestration")
		return nil, err
	}

	no, ok := mapOfNestedOrcs[ea.definition.OrchestrationId]
	if !ok {
		err = fmt.Errorf("unknown loop body orchestration id %s", ea.definition.OrchestrationId)
		return nil, err
	}

	ea.bodyOrchestration = no
	return ea, nil
}

func (a *LoopActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.LoopActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.LoopActivityType)).Msg("activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.LoopActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.LoopActivityType)
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

	evaluator, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", false)
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}

	controlFlow, err := InitLoopControlFlow(a.definition.ControlFlow, evaluator)
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func() { a.SetMetrics(beginOf, metricsLabels) }()

	req, err := a.newRequestDefinition(wfc, expressionCtx)
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		metricsLabels[MetricIdStatusCode] = "500"
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	_ = wfc.SetHarEntryRequest(a.Name(), req, config.PersonallyIdentifiableInformation{})

	st := http.StatusOK

	var loopBodyResponses [][]byte
	var activityError error
	var loopBodyStatusCode int
	for controlFlow.HasNext(wfc) {

		log.Info().Int("loop", controlFlow.current).Msg(semLogContext)
		var loopBodyInputRequest []byte
		loopBodyInputRequest, err = controlFlow.Next(wfc, evaluator)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			metricsLabels[MetricIdStatusCode] = "500"
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		wfcChild, err := wfc.NewChild(
			expressionCtx,
			a.bodyOrchestration.Cfg.Id,
			a.bodyOrchestration.Cfg.Version,
			a.bodyOrchestration.Cfg.SHA,
			a.bodyOrchestration.Cfg.Description,
			a.bodyOrchestration.Cfg.Dictionaries,
			a.bodyOrchestration.Cfg.References,
			tcfg.ProcessVars,
			loopBodyInputRequest,
			nil)

		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			metricsLabels[MetricIdStatusCode] = "500"
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		wfcChild.RequestDeadline = a.bodyOrchestration.Cfg.GetPropertyAsDuration(config.OrchestrationPropertyRequestDeadline, time.Duration(0))
		if wfcChild.RequestDeadline != 0 {
			log.Info().Float64("deadline-secs", wfcChild.RequestDeadline.Seconds()).Msg(semLogContext + " - setting workflow case request deadline")
		} else {
			log.Info().Msg(semLogContext + " - no workflow case request deadline has been set")
		}

		err = a.executeNestedOrchestration(wfcChild)
		harData := wfcChild.GetHarData(wfcase.ReportLogHAR, nil)
		if harData != nil {
			entryId := wfc.ComputeFirstAvailableIndexedHarEntryId(a.Name() + "-body")
			for _, e := range harData.Log.Entries {
				if strings.HasPrefix(e.Comment, "request") {
					e.Comment = fmt.Sprintf("%s", entryId)
					if e.Response != nil && e.Response.Content != nil {
						if strings.HasPrefix(e.Response.Content.MimeType, constants.ContentTypeApplicationJson) {
							loopBodyResponses = append(loopBodyResponses, e.Response.Content.Data)
						} else {
							log.Warn().Msg(semLogContext + " non application/json response")
						}
					}
				} else {
					e.Comment = fmt.Sprintf("%s@%s", entryId, e.Comment)
				}
				wfc.Entries[e.Comment] = e
			}
		}

		st = http.StatusOK
		if err != nil {
			st = http.StatusInternalServerError
			var smpErr *smperror.SymphonyError
			if errors.As(err, &smpErr) {
				st = smpErr.StatusCode
			}
		}

		loopBodyStatusCode, err = a.ProcessResponseActionByStatusCode(st, a.Name(), a.Name(), wfc, wfcChild, wfcase.HarEntryReference{Name: "request", UseResponse: true}, a.definition.OnResponseActions, false)
		if err != nil {
			activityError = err
			break
		}
	}

	var harResponse *har.Response
	if activityError == nil {
		harResponse, _ = a.newSuccessResponse(loopBodyResponses)
	} else {
		harResponse, _ = a.newErrorResponse(st, activityError)
	}

	if harResponse != nil {
		_ = wfc.SetHarEntryResponse(a.Name(), harResponse, tcfg.PII)
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(harResponse.Status)
	}

	// remappedStatusCode, err := a.ProcessResponseActionByStatusCode(st, a.Name(), a.Name(), wfc, nil, wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}, a.definition.OnResponseActions, true)
	if loopBodyStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(loopBodyStatusCode)
	}
	if activityError != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), activityError)
		return smperror.NewExecutableServerError(smperror.WithError(activityError), smperror.WithErrorAmbit(a.Name()))
	}

	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return nil
}

func (a *LoopActivity) executeNestedOrchestration(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.LoopActivityType) + "::execute-orchestration"

	var orchestrationErr error

	var harResponse *har.Response
	var finalExec executable.Executable
	finalExec, orchestrationErr = a.bodyOrchestration.Execute(wfc)
	if orchestrationErr == nil {
		respExec, isResponseActivity := finalExec.(*responseactivity.ResponseActivity)
		if !isResponseActivity {
			log.Fatal().Err(errors.New("final activity is not a response activity")).Msg(semLogContext)
		}

		var resp *har.Response
		resp, orchestrationErr = respExec.ResponseJSON(wfc)
		if orchestrationErr == nil {
			harResponse = har.NewResponse(
				resp.Status, resp.StatusText,
				resp.Content.MimeType, resp.Content.Data,
				resp.Headers,
			)
			_ = wfc.SetHarEntryResponse("request", harResponse, a.bodyOrchestration.Cfg.PII)
			log.Info().Str("response", string(resp.Content.Data)).Msg(semLogContext)
		}
	}

	if orchestrationErr != nil {
		log.Error().Err(orchestrationErr).Msg(semLogContext)
		sc, ct, resp := produceLoopActivityErrorResponse(orchestrationErr)
		harResponse = har.NewResponse(
			sc, "execution error",
			constants.ContentTypeApplicationJson, resp,
			[]har.NameValuePair{{Name: constants.ContentTypeHeader, Value: ct}},
		)
		err := wfc.SetHarEntryResponse("request", harResponse, a.bodyOrchestration.Cfg.PII)
		if err != nil {
			log.Error().Err(err).Str("response", string(resp)).Msg(semLogContext)
		} else {
			log.Error().Str("response", string(resp)).Msg(semLogContext)
		}
	}

	bndry, ok := a.bodyOrchestration.Cfg.FindBoundaryByName(config.DefaultActivityBoundary)
	if ok {
		err := a.bodyOrchestration.ExecuteBoundary(wfc, bndry)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}

		orchestrationErr = util.CoalesceError(orchestrationErr, err)
	}

	return orchestrationErr
}

func produceLoopActivityErrorResponse(err error) (int, string, []byte) {
	var exeErr *smperror.SymphonyError
	ok := errors.As(err, &exeErr)

	if !ok {
		exeErr = smperror.NewExecutableServerError(smperror.WithStep("not-applicable"), smperror.WithErrorAmbit("general"), smperror.WithErrorMessage(err.Error()))
	}

	response, err := exeErr.ToJSON(nil)
	if err != nil {
		response = []byte(fmt.Sprintf("{ \"err\": %s }", util.JSONEscape(err.Error(), false)))
	}

	return exeErr.StatusCode, constants.ContentTypeApplicationJson, response
}

//const (
//	MetricIdActivityType = "type"
//	MetricIdActivityName = "name"
//	MetricIdOpType       = "op-type"
//	MetricIdStatusCode   = "status-code"
//)

func (a *LoopActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}

func (a *LoopActivity) newRequestDefinition(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference) (*har.Request, error) {

	const semLogContext = string(config.LoopActivityType) + "::new-request-definition"

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(0000)
	ub.WithScheme("activity")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s", string(config.LoopActivityType), a.Name()))
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

func (a *LoopActivity) newErrorResponse(st int, err error) (*har.Response, error) {

	b := []byte(err.Error())

	var r *har.Response
	r = &har.Response{
		Status:      st,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(st),
		HeadersSize: -1,
		BodySize:    int64(len(b)),
		Headers:     []har.NameValuePair{},
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeTextPlain,
			Size:     int64(len(b)),
			Data:     b,
		},
	}

	return r, nil
}

func (a *LoopActivity) newSuccessResponse(loopResponses [][]byte) (*har.Response, error) {

	var sb bytes.Buffer
	sb.WriteString("[")
	for i, resp := range loopResponses {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(string(resp))
	}
	sb.WriteString("]")
	b := sb.Bytes()

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
