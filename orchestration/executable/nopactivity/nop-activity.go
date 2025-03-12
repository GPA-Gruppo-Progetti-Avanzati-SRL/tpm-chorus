package nopactivity

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/rs/zerolog/log"
	"time"
)

type NopActivity struct {
	executable.Activity
}

func NewNopActivity(item config.Configurable, refs config.DataReferences) (*NopActivity, error) {
	var err error

	ea := &NopActivity{}
	ea.Cfg = item
	ea.Refs = refs

	_, ok := item.(*config.NopActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", item, config.NopActivityType)
		return nil, err
	}

	return ea, nil
}

func (a *NopActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.NopActivityType) + "::execute"
	var err error
	if !a.IsEnabled(wfc) {
		log.Info().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.EchoActivityType)).Msg("activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.NopActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.NopActivityType)
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

	return nil
}
