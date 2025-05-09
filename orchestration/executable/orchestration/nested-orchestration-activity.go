package orchestration

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/responseactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

type NestedOrchestrationActivity struct {
	executable.Activity
	definition    config.NestedOrchestrationActivityDefinition
	orchestration Orchestration
}

func NewNestedOrchestrationActivity(item config.Configurable, refs config.DataReferences, mapOfNestedOrcs map[string]Orchestration) (*NestedOrchestrationActivity, error) {
	var err error

	ea := &NestedOrchestrationActivity{}
	ea.Cfg = item
	ea.Refs = refs

	eaCfg, ok := item.(*config.NestedOrchestrationActivity)
	if !ok {
		err := fmt.Errorf("this is weird %T is not %s config type", item, config.NestedOrchestrationActivityType)
		return nil, err
	}

	ea.definition, err = config.UnmarshalNestedOrchestrationActivityDefinition(eaCfg.Definition, refs)
	if err != nil {
		return nil, err
	}

	if ea.definition.OrchestrationId == "" {
		err = errors.New("nested orchestration must specify id of orchestration")
		return nil, err
	}

	no, ok := mapOfNestedOrcs[ea.definition.OrchestrationId]
	if !ok {
		err = fmt.Errorf("unknown nested orchestration id %s", ea.definition.OrchestrationId)
		return nil, err
	}

	ea.orchestration = no
	return ea, nil
}

func (a *NestedOrchestrationActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.NestedOrchestrationActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.NestedOrchestrationActivityType)).Msg("activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.NestedOrchestrationActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.NestedOrchestrationActivityType)
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

	/*
		Vars are used as params...
		if len(tcfg.ProcessVars) > 0 {
			err = wfc.SetVars(expressionCtx, tcfg.ProcessVars, "", false)
			if err != nil {
				wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
				return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
			}
		}
	*/

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func() { a.SetMetrics(beginOf, metricsLabels) }()

	wfcChild, err := wfc.NewChild(
		expressionCtx,
		a.orchestration.Cfg.Id,
		a.orchestration.Cfg.Version,
		a.orchestration.Cfg.SHA,
		a.orchestration.Cfg.Description,
		a.orchestration.Cfg.Dictionaries,
		a.orchestration.Cfg.References,
		tcfg.ProcessVars,
		nil,
		nil)

	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		metricsLabels[MetricIdStatusCode] = "500"
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	wfcChild.RequestDeadline = a.orchestration.Cfg.GetPropertyAsDuration(config.OrchestrationPropertyRequestDeadline, time.Duration(0))
	if wfcChild.RequestDeadline != 0 {
		log.Info().Float64("deadline-secs", wfcChild.RequestDeadline.Seconds()).Msg(semLogContext + " - setting workflow case request deadline")
	} else {
		log.Info().Msg(semLogContext + " - no workflow case request deadline has been set")
	}

	var st int
	st, err = a.executeNestedOrchestration(wfcChild)
	harData := wfcChild.GetHarData(wfcase.ReportLogHAR, nil)
	if harData != nil {
		entryId := wfc.ComputeFirstAvailableIndexedHarEntryId(a.Name())
		for _, e := range harData.Log.Entries {
			if strings.HasPrefix(e.Comment, "request") {
				e.Comment = fmt.Sprintf("%s", entryId)
			} else {
				e.Comment = fmt.Sprintf("%s@%s", entryId, e.Comment)
			}
			wfc.Entries[e.Comment] = e
		}
	}

	if err != nil {
		st = http.StatusInternalServerError
	}

	remappedStatusCode, err := a.ProcessResponseActionByStatusCode(st, a.Name(), a.Name(), wfc, wfcChild, wfcase.HarEntryReference{Name: "request", UseResponse: true}, a.definition.OnResponseActions, false)
	if remappedStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
	}
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithError(err))
	}

	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return nil
}

func (a *NestedOrchestrationActivity) executeNestedOrchestration(wfc *wfcase.WfCase) (int, error) {
	const semLogContext = string(config.NestedOrchestrationActivityType) + "::execute-orchestration"
	nestedStatusCode := http.StatusInternalServerError

	var orchestrationErr error

	var harResponse *har.Response
	var finalExec executable.Executable
	finalExec, orchestrationErr = a.orchestration.Execute(wfc)
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
			_ = wfc.SetHarEntryResponse("request", harResponse, a.orchestration.Cfg.PII)
			log.Info().Str("response", string(resp.Content.Data)).Msg(semLogContext)
			nestedStatusCode = resp.Status
		}
	}

	if orchestrationErr != nil {
		log.Error().Err(orchestrationErr).Msg(semLogContext)
		sc, ct, resp := produceNestedActivityErrorResponse(orchestrationErr)
		harResponse = har.NewResponse(
			sc, "execution error",
			constants.ContentTypeApplicationJson, resp,
			[]har.NameValuePair{{Name: constants.ContentTypeHeader, Value: ct}},
		)
		err := wfc.SetHarEntryResponse("request", harResponse, a.orchestration.Cfg.PII)
		if err != nil {
			log.Error().Err(err).Str("response", string(resp)).Msg(semLogContext)
		} else {
			log.Error().Str("response", string(resp)).Msg(semLogContext)
		}
		nestedStatusCode = sc
	}

	bndry, ok := a.orchestration.Cfg.FindBoundaryByName(config.DefaultActivityBoundary)
	if ok {
		err := a.orchestration.ExecuteBoundary(wfc, bndry)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}

		orchestrationErr = util.CoalesceError(orchestrationErr, err)
	}

	return nestedStatusCode, orchestrationErr
}

func produceNestedActivityErrorResponse(err error) (int, string, []byte) {
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

const (
	MetricIdActivityType = "type"
	MetricIdActivityName = "name"
	MetricIdOpType       = "op-type"
	MetricIdStatusCode   = "status-code"
)

func (a *NestedOrchestrationActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}
