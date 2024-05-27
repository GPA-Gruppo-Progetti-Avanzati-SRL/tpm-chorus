package transform

import (
	"errors"
	"fmt"
	"github.com/qntfy/kazaam"
	"github.com/rs/zerolog/log"
	"tpm-chorus/orchestration/transform/operators"
)

var kc kazaam.Config

type Transformation struct {
	Cfg    Config
	Kazaam *kazaam.Kazaam
}

type Registry map[string]Transformation

var registry Registry

func InitializeKazaamRegistry() error {
	kc = kazaam.NewDefaultConfig()

	err := kc.RegisterTransform(OperatorShiftArrayItems, operators.ShiftArrayItems(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(OperatorNoOp, operators.NoOp(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(OperatorFormat, operators.Format(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(OperatorFilterArrayItems, operators.FilterArrayItems(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(OperatorReduceArray, operators.ReduceArray(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(OperatorSetProperties, operators.SetProperties(kc))
	if err != nil {
		return err
	}

	registry = make(map[string]Transformation)
	return nil
}

func GetRegistry() Registry {
	return registry
}

func (r Registry) Add(tcfg Config) error {
	if tcfg.Id == "" {
		err := errors.New("transformation require an id")
		return err
	}

	if _, ok := r[tcfg.Id]; ok {
		err := fmt.Errorf("transformation is must be unique (conflicting id: %s)", tcfg.Id)
		return err
	}

	rule, err := tcfg.ToJSONRule()
	if err != nil {
		return err
	}

	k, err := kazaam.New(string(rule), kc)

	r[tcfg.Id] = Transformation{Cfg: tcfg, Kazaam: k}
	return nil
}

func (r Registry) Get(id string) (Transformation, error) {
	if id == "" {
		err := errors.New("transformation require an id")
		return Transformation{}, err
	}

	t, ok := r[id]
	if !ok {
		err := fmt.Errorf("transformation not found (missing id: %s)", id)
		return Transformation{}, err
	}

	return t, nil
}

func (r Registry) Transform(id string, data []byte) ([]byte, error) {
	const semLogContext = "transform-registry::transform"
	log.Debug().Str("id", id).Msg(semLogContext)
	t, err := r.Get(id)
	if err != nil {
		return nil, err
	}

	if t.Cfg.Verbose {
		log.Trace().Str("id", t.Cfg.Id).Str("input", string(data)).Msg(semLogContext)
	}
	dataOut, err := t.Kazaam.TransformJSONString(string(data))
	if t.Cfg.Verbose {
		if err != nil {
			log.Error().Err(err).Str("id", t.Cfg.Id).Msg(semLogContext)
		} else {
			log.Trace().Str("id", t.Cfg.Id).Str("output", string(data)).Msg(semLogContext)
		}
	}

	return dataOut, err
}
