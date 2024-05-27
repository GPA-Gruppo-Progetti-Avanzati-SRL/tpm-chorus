package echoactivity

import (
	"github.com/rs/zerolog/log"
	"tpm-chorus/constants"
	"tpm-chorus/orchestration/config"
	"tpm-chorus/orchestration/executable"
	"tpm-chorus/orchestration/wfcase"
	"tpm-chorus/smperror"
)

type EchoActivity struct {
	executable.Activity
}

func NewEchoActivity(item config.Configurable, refs config.DataReferences) (*EchoActivity, error) {

	ea := &EchoActivity{}
	ea.Cfg = item
	ea.Refs = refs
	return ea, nil
}

func (a *EchoActivity) Execute(wfc *wfcase.WfCase) error {

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "echo").Msg("activity not enabled")
		return nil
	}

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "echo").Msg("start activity")

	tcfg, ok := a.Cfg.(*config.EchoActivity)
	if !ok {
		log.Error().Msgf("this is weird %v is not (*config.EchoActivity)", a.Cfg)
	}

	if len(tcfg.ProcessVars) > 0 {
		err := wfc.SetVars("request", tcfg.ProcessVars, "", false)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "echo").Msg(tcfg.Message)
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return nil
}
