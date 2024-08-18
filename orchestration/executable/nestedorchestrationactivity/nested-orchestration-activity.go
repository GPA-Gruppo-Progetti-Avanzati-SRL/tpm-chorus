package nestedorchestrationactivity

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/rs/zerolog/log"
)

type NestedOrchestrationActivity struct {
	executable.Activity
}

func NewNestedOrchestrationActivity(item config.Configurable, refs config.DataReferences) (*NestedOrchestrationActivity, error) {

	ea := &NestedOrchestrationActivity{}
	ea.Cfg = item
	ea.Refs = refs
	return ea, nil
}

func (a *NestedOrchestrationActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.NestedOrchestrationActivityType) + "::execute"
	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.NestedOrchestrationActivityType)).Msg("activity not enabled")
		return nil
	}

	expressionCtx, err := wfc.ResolveExpressionContextName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}
	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext + " start")

	tcfg, ok := a.Cfg.(*config.NestedOrchestrationActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.NestedOrchestrationActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}
	if len(tcfg.ProcessVars) > 0 {
		err := wfc.SetVars(expressionCtx, tcfg.ProcessVars, "", false)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("msg", tcfg.Message).Msg(semLogContext + " end")
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return nil
}
