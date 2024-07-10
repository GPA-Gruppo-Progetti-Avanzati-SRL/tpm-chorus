package requestactivity

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"time"
)

const (
	MetricIdActivityType = "type"
	MetricIdActivityName = "name"
	MetricIdStatusCode   = "status-code"
)

type RequestActivity struct {
	executable.Activity
}

func NewRequestActivity(item config.Configurable, refs config.DataReferences) (*RequestActivity, error) {

	a := &RequestActivity{}
	a.Cfg = item
	a.Refs = refs
	return a, nil
}

func (a *RequestActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.RequestActivityType) + "::execute"

	var err error
	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func(start time.Time) {
		a.SetMetrics(start, metricsLabels)
	}(beginOf)

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "request").Msg(semLogContext + " start activity")
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)

	cfg, ok := a.Cfg.(*config.RequestActivity)
	if !ok {
		err := fmt.Errorf("not a proper config for request activity %v", a.Cfg)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	// if len(cfg.ProcessVars) > 0 {
	err = wfc.SetVars(wfcase.InitialRequestResolverContext, cfg.ProcessVars, "", false)
	if err != nil {
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}
	// }

	if len(cfg.Validations) > 0 {
		for _, v := range cfg.Validations {
			b, err := wfc.Vars.BoolEval(v.Expr)
			if err != nil {
				return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
			}

			if !b {

				// 20220726 Handle the case where some values have not been set. In case produce same values previously generated. Expr is required.
				cd := ""
				if v.Name != "" {
					cd = v.Name
				}

				msg := "failed inter-field validation"
				if v.Description != "" {
					msg = v.Description
				}
				return smperror.NewBadRequestError(smperror.WithCode(cd), smperror.WithErrorMessage(msg))
			}
		}
	}

	metricsLabels[MetricIdStatusCode] = fmt.Sprint(200)
	return nil
}

func (a *RequestActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "500",
	}

	return metricsLabels
}
