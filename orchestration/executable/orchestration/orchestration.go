package orchestration

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/echoactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/endpointactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/kafkactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/requestactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/responseactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/rs/zerolog/log"
)

type Orchestration struct {
	Cfg         *config.Orchestration
	Executables map[string]executable.Executable
}

func NewOrchestration(cfg *config.Orchestration) (Orchestration, error) {

	o := Orchestration{Cfg: cfg}
	var execs map[string]executable.Executable

	for _, cfgItem := range cfg.Activities {

		var ex executable.Executable
		var err error
		switch cfgItem.Type() {
		case config.RequestActivityType:
			ex, err = requestactivity.NewRequestActivity(cfgItem, cfg.References)
		case config.EchoActivityType:
			ex, err = echoactivity.NewEchoActivity(cfgItem, cfg.References)
		case config.ResponseActivityType:
			ex, err = responseactivity.NewResponseActivity(cfgItem, cfg.References)
		case config.EndpointActivityType:
			ex, err = endpointactivity.NewEndpointActivity(cfgItem, cfg.References)
		case config.KafkaActivityType:
			ex, err = kafkactivity.NewKafkaActivity(cfgItem, cfg.References)
		default:
			panic(fmt.Errorf("this should not happen %s, unrecognized sctivity type", cfgItem.Type()))
		}

		if err != nil {
			return o, err
		}

		if execs == nil {
			execs = make(map[string]executable.Executable)
		}

		execs[cfgItem.Name()] = ex
	}

	if len(execs) == 0 {
		return o, errors.New("empty orchestration found")
	}
	o.Executables = execs

	for _, pcfg := range cfg.Paths {
		p, err := executable.NewPath(pcfg)
		if err != nil {
			return o, err
		}

		var ex executable.Executable
		var ok bool
		if ex, ok = execs[pcfg.SourceName]; !ok {
			return o, fmt.Errorf("dangling path, could not find source %s", pcfg.SourceName)
		}

		ex.AddOutput(p)

		if ex, ok = execs[pcfg.TargetName]; !ok {
			return o, fmt.Errorf("dangling path, could not find target %s", pcfg.TargetName)
		}

		ex.AddInput(p)
	}

	return o, nil
}

func (o *Orchestration) IsValid() bool {

	if len(o.Executables) == 0 {
		log.Trace().Msg("empty orchestration found")
		return false
	}

	rc := true
	sa := ""
	for _, ex := range o.Executables {
		if !ex.IsValid() {
			rc = false
		}

		if ex.Type() == config.RequestActivityType {
			sa = ex.Name()
		}
	}

	if sa == "" {
		log.Trace().Msg("start activity incorrectly set")
		return false
	}

	if o.Cfg.StartActivity == "" {
		log.Info().Msg("start activity not set in config...fixing")
		o.Cfg.StartActivity = sa
	} else if sa != o.Cfg.StartActivity {
		log.Trace().Msg("start activity mismatch ")
		return false
	}

	return rc
}

func (o *Orchestration) Execute(wfc *wfcase.WfCase) (executable.Executable, error) {

	const semLogContext = "orchestration::execute"
	na := o.Cfg.StartActivity
	var a executable.Executable
	currentBoundary := config.DefaultActivityBoundary
	for na != "" {
		a = o.Executables[na]
		if a.Boundary() != currentBoundary {
			log.Info().Str("current-boundary", currentBoundary).Str("next-boundary", a.Boundary()).Msg(semLogContext + " boundary limit")
		}

		err := a.Execute(wfc)
		if err != nil {
			return a, err
		}

		na, err = a.Next(wfc)
		if err != nil {
			return a, err
		}
	}

	return a, nil
}

func (o *Orchestration) ExecuteBoundary(wfc *wfcase.WfCase, boundary config.ExecBoundary) error {

	const semLogContext = "orchestration::execute-boundary"

	log.Trace().Str("boundary", boundary.Name).Msg(semLogContext)
	withError := false
	for _, na := range boundary.Activities {
		a := o.Executables[na]
		err := a.Execute(wfc)
		if err != nil {
			withError = true
			log.Error().Err(err).Msg(semLogContext)
		}
	}

	if withError {
		return errors.New("boundary errors")
	}
	return nil
}
