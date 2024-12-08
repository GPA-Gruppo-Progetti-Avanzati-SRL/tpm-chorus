package executable

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"time"
)

type Executable interface {
	Execute(wfc *wfcase.WfCase) error
	Next(wfc *wfcase.WfCase, policy string) (string, error)
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

	return wfc.EvalBoolExpression(a.Cfg.Enabled())
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

func (a *Activity) Next(wfc *wfcase.WfCase, policy string) (string, error) {

	na := ""
	if len(a.Outputs) > 0 {
		outputVect := make([]string, 0)
		for _, v := range a.Outputs {
			outputVect = append(outputVect, v.Cfg.Constraint)
		}
		selectedPath, err := wfc.EvalBoolExpressionSet(outputVect, policy)
		if err != nil {
			return "", smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
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

func (a *Activity) GetEvaluator(wfc *wfcase.WfCase) (*wfexpressions.Evaluator, error) {
	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		return nil, err
	}

	resolver, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", false)
	if err != nil {
		return nil, err
	}

	return resolver, nil
}

func (a *Activity) ProcessResponseActionByStatusCode(
	st int,
	ambitName, stepName string,
	destWfc *wfcase.WfCase,
	srcWfc *wfcase.WfCase,
	contextReference wfcase.HarEntryReference,
	actions config.OnResponseActions,
	ignoreNonJSONResponseContent bool) (int, error) {

	actNdx := actions.FindByStatusCode(st)
	if actNdx < 0 {
		return -1, nil
	}

	if !ignoreNonJSONResponseContent {
		ignoreNonJSONResponseContent = actions[actNdx].IgnoreNonApplicationJsonResponseContent
	}

	if srcWfc == nil {
		srcWfc = destWfc
	}

	act := actions[actNdx]
	transformId, err := a.ChooseTransformation(srcWfc, act.Transforms)
	if err != nil {
		log.Error().Err(err).Str("ctx", stepName).Str("request-id", srcWfc.GetRequestId()).Msg("processResponseAction: error in selecting transformation")
		return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambitName), smperror.WithStep(stepName), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
	}

	//contextReference := wfcase.ResolverContextReference{Name: a.Name(), UseResponse: true}

	if len(act.ProcessVars) > 0 {
		err = destWfc.SetVarsFromCase(srcWfc, contextReference, act.ProcessVars, transformId, ignoreNonJSONResponseContent)
		if err != nil {
			log.Error().Err(err).Str("ctx", stepName).Str("request-id", srcWfc.GetRequestId()).Msg("processResponseAction: error in setting variables")
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambitName), smperror.WithStep(stepName), smperror.WithCode("500"), smperror.WithErrorMessage("error processing response body"), smperror.WithDescription(err.Error()))
		}
	}

	if ndx := a.ChooseError(srcWfc, act.Errors); ndx >= 0 {

		e := act.Errors[ndx]
		ambit := e.Ambit
		if ambit == "" {
			ambit = ambitName
		}

		step := e.Step
		if step == "" {
			step = stepName
		}

		statusCode := st
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}

		m, err := srcWfc.ResolveStrings(contextReference, []string{e.Code, e.Message, e.Description, step}, "", ignoreNonJSONResponseContent)
		if err != nil {
			log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
		}
		return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]), smperror.WithLevel(e.ErrorLevel))
	}

	return 0, nil
}

func (a *Activity) ChooseTransformation(wfc *wfcase.WfCase, trs []kzxform.TransformReference) (string, error) {
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

func (a *Activity) ChooseError(wfc *wfcase.WfCase, errors []config.ErrorInfo) int {
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
