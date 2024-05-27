package executable

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"time"
	"tpm-chorus/constants"
	"tpm-chorus/orchestration/config"
	"tpm-chorus/orchestration/wfcase"
	error2 "tpm-chorus/smperror"
)

type Executable interface {
	Execute(wfc *wfcase.WfCase) error
	Next(wfc *wfcase.WfCase) (string, error)
	AddInput(p Path) error
	AddOutput(p Path) error
	IsValid() bool
	Type() config.Type
	Name() string
	Boundary() string
	IsEnabled(wfc *wfcase.WfCase) bool
}

type Activity struct {
	Cfg     config.Configurable
	Refs    config.DataReferences
	Outputs []Path
	Inputs  []Path
}

func (a *Activity) Name() string {
	return a.Cfg.Name()
}

func (a *Activity) Type() config.Type {
	return a.Cfg.Type()
}

func (a *Activity) Boundary() string {
	return a.Cfg.Boundary()
}

func (a *Activity) AddOutput(p Path) error {
	a.Outputs = append(a.Outputs, p)
	return nil
}

func (a *Activity) AddInput(p Path) error {
	a.Inputs = append(a.Inputs, p)
	return nil
}

func (a *Activity) IsEnabled(wfc *wfcase.WfCase) bool {

	if a.Cfg.Enabled() == "" {
		return true
	}

	return wfc.EvalExpression(a.Cfg.Enabled())
}

func (a *Activity) IsValid() bool {

	const semLogContext = "activity::is-valid"

	rc := true

	if a.Cfg.IsBoundary() {
		if len(a.Inputs) != 0 || len(a.Outputs) != 0 {
			log.Trace().Str(constants.SemLogActivityName, a.Cfg.Name()).Int(constants.SemLogActivityNumOutputs, len(a.Outputs)).Int(constants.SemLogActivityNumInputs, len(a.Inputs)).Msg(semLogContext + " boundary activity must not have connections")
			return false
		}

		return true
	}

	switch a.Cfg.Type() {
	case config.RequestActivityType:
		if len(a.Outputs) == 0 {
			log.Trace().Str(constants.SemLogActivityName, a.Cfg.Name()).Int(constants.SemLogActivityNumOutputs, len(a.Outputs)).Msg("start activity missing outputs")
			rc = false
		}

		if len(a.Inputs) != 0 {
			log.Trace().Str(constants.SemLogActivityName, a.Cfg.Name()).Int(constants.SemLogActivityNumInputs, len(a.Inputs)).Msg("start activity doesn't have inputs")
			rc = false
		}

	case config.ResponseActivityType:
		if len(a.Inputs) == 0 {
			log.Trace().Str(constants.SemLogActivityName, a.Cfg.Name()).Int(constants.SemLogActivityNumInputs, len(a.Inputs)).Msg("end activity missing inputs")
			rc = false
		}

		if len(a.Outputs) != 0 {
			log.Trace().Str(constants.SemLogActivityName, a.Cfg.Name()).Int(constants.SemLogActivityNumOutputs, len(a.Outputs)).Msg("end activity doesn't have outputs")
			rc = false
		}

	default:
		if len(a.Inputs) == 0 || len(a.Outputs) == 0 {
			log.Trace().Str(constants.SemLogActivityName, a.Cfg.Name()).Int(constants.SemLogActivityNumOutputs, len(a.Outputs)).Int(constants.SemLogActivityNumInputs, len(a.Inputs)).Msg("activity missing connections")
			rc = false
		}
	}

	return rc
}

func (a *Activity) Next(wfc *wfcase.WfCase) (string, error) {

	na := ""
	if len(a.Outputs) > 0 {
		outputVect := make([]string, 0)
		for _, v := range a.Outputs {
			outputVect = append(outputVect, v.Cfg.Constraint)
		}
		selectedPath, err := wfc.BooleanEvalProcessVars(outputVect)
		if err != nil {
			return "", error2.NewExecutableServerError(error2.WithErrorAmbit(a.Name()), error2.WithErrorMessage(err.Error()))
		}
		return a.Outputs[selectedPath].Cfg.TargetName, nil
	}
	log.Trace().Str(constants.SemLogNextActivity, na).Str(constants.SemLogActivity, a.Name()).Msg("next activity execution")
	return na, nil
}

func (a *Activity) MetricsGroup() (promutil.Group, bool, error) {
	mCfg := a.Cfg.MetricsConfig()

	var g promutil.Group
	var err error
	var ok bool
	if mCfg.IsEnabled() {
		g, err = promutil.GetGroup(mCfg.GId)
		if err == nil {
			ok = true
		}
	}

	return g, ok, err
}

func (a *Activity) SetMetrics(begin time.Time, lbls prometheus.Labels) error {

	const semLogContext = "executable::set-metrics"
	cfg := a.Cfg.MetricsConfig()
	if cfg.IsEnabled() {
		g, _, err := a.MetricsGroup()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		if cfg.IsCounterEnabled() {
			g.SetMetricValueById(cfg.CounterId, 1, lbls)
		}

		if cfg.IsHistogramEnabled() {
			g.SetMetricValueById(cfg.HistogramId, time.Since(begin).Seconds(), lbls)
		}
	}

	return nil
}
