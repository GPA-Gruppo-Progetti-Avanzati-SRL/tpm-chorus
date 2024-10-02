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
	"github.com/rs/zerolog/log"
)

type NestedOrchestrationActivity struct {
	executable.Activity
	orchestration Orchestration
}

func NewNestedOrchestrationActivity(item config.Configurable, refs config.DataReferences, orc Orchestration) (*NestedOrchestrationActivity, error) {

	ea := &NestedOrchestrationActivity{}
	ea.Cfg = item
	ea.Refs = refs
	ea.orchestration = orc
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

	expressionCtx, err := wfc.ResolveExpressionContextName(a.Cfg.ExpressionContextNameStringReference())
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

	wfcChild, err := wfc.NewChild(
		expressionCtx,
		a.orchestration.Cfg.Id,
		a.orchestration.Cfg.Version,
		a.orchestration.Cfg.SHA,
		a.orchestration.Cfg.Description,
		a.orchestration.Cfg.Dictionaries,
		a.orchestration.Cfg.References,
		tcfg.ProcessVars,
		nil)

	err = a.executeNestedOrchestration(wfcChild)
	harData := wfcChild.GetHarData(wfcase.ReportLogHAR, nil)
	if harData != nil {
		for _, e := range harData.Log.Entries {
			if e.Comment == "request" {
				e.Comment = fmt.Sprintf("%s", a.Name())
			} else {
				e.Comment = fmt.Sprintf("%s#%s", a.Name(), e.Comment)
			}
			wfc.Entries[e.Comment] = e
		}
	}

	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return nil
}

func (a *NestedOrchestrationActivity) executeNestedOrchestration(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.NestedOrchestrationActivityType) + "::execute-orchestration"

	var orchestrationErr error

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
			_ = wfc.AddEndpointResponseData(
				"request",
				har.NewResponse(
					resp.Status, resp.StatusText,
					resp.Content.MimeType, resp.Content.Data,
					resp.Headers,
				),
				a.orchestration.Cfg.PII)
			log.Info().Str("response", string(resp.Content.Data)).Msg(semLogContext)
		}
	}

	if orchestrationErr != nil {
		log.Error().Err(orchestrationErr).Msg(semLogContext)
		sc, ct, resp := produceErrorResponse(orchestrationErr)
		err := wfc.AddEndpointResponseData(
			"request",
			har.NewResponse(
				sc, "execution error",
				constants.ContentTypeApplicationJson, resp,
				[]har.NameValuePair{{Name: constants.ContentTypeHeader, Value: ct}},
			),
			a.orchestration.Cfg.PII)
		if err != nil {
			log.Error().Err(err).Str("response", string(resp)).Msg(semLogContext)
		} else {
			log.Error().Str("response", string(resp)).Msg(semLogContext)
		}
	}

	bndry, ok := a.orchestration.Cfg.FindBoundaryByName(config.DefaultActivityBoundary)
	if ok {
		err := a.orchestration.ExecuteBoundary(wfc, bndry)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}

		orchestrationErr = util.CoalesceError(orchestrationErr, err)
	}

	return orchestrationErr
}

func produceErrorResponse(err error) (int, string, []byte) {
	var exeErr *smperror.SymphonyError
	ok := errors.As(err, &exeErr)

	if !ok {
		exeErr = smperror.NewExecutableServerError(smperror.WithStep("not-applicable"), smperror.WithErrorAmbit("general"), smperror.WithErrorMessage(err.Error()))
	}

	response, err := exeErr.ToJSON(nil)
	if err != nil {
		response = []byte(fmt.Sprintf("{ \"err\": %s }", util.JSONEscape(err.Error())))
	}

	return exeErr.StatusCode, constants.ContentTypeApplicationJson, response
}