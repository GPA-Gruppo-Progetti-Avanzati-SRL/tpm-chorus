package kzxform

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/addarrayitems"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/distinctarrayitems"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/filterarrayitems"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/format"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/lenarrays"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/mergearrays"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/reducearray"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/setproperties"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/shiftarrayitems"
	"github.com/qntfy/kazaam"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
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

	err := kc.RegisterTransform(shiftarrayitems.OperatorShiftArrayItems, shiftarrayitems.ShiftArrayItems(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(operators.OperatorNoOp, operators.NoOp(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(format.OperatorFormat, format.Format(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(filterarrayitems.OperatorFilterArrayItems, filterarrayitems.FilterArrayItems(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(reducearray.OperatorReduceArray, reducearray.ReduceArray(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(setproperties.OperatorSetProperties, setproperties.SetProperties(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(lenarrays.OperatorLenArrays, lenarrays.LenArrays(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(mergearrays.OperatorMergeArrays, mergearrays.MergeArrays(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(addarrayitems.OperatorAddArrayItems, addarrayitems.AddArrayItems(kc))
	if err != nil {
		return err
	}

	err = kc.RegisterTransform(distinctarrayitems.OperatorDistinctArrayItems, distinctarrayitems.DistinctArrayItems(kc))
	if err != nil {
		return err
	}

	registry = make(map[string]Transformation)
	return nil
}

func GetRegistry() Registry {

	const semLogContext = "transform-registry::get"

	if registry == nil {
		err := InitializeKazaamRegistry()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}
	}

	return registry
}

func (r Registry) AddTransformation(ref TransformReference) error {
	trsf := Config{}
	err := yaml.Unmarshal(ref.Data, &trsf)
	if err != nil {
		return err
	}

	// Force the id to the provided one.
	trsf.Id = ref.Id
	err = r.Add3(trsf)
	if err != nil {
		return err
	}

	return nil
}

func (r Registry) Add3(tcfg Config) error {

	const semLogContext = "transform-registry::add"
	if tcfg.Id == "" {
		err := errors.New("transformation require an id")
		return err
	}

	if _, ok := r[tcfg.Id]; ok {
		err := fmt.Errorf("transformation id must be unique (conflicting id: %s)", tcfg.Id)
		log.Warn().Err(err).Msg(semLogContext)
		return nil
	}

	rule, err := tcfg.ToJSONRule()
	if err != nil {
		return err
	}

	k, err := kazaam.New(string(rule), kc)

	r[tcfg.Id] = Transformation{Cfg: tcfg, Kazaam: k}
	return nil
}

var XFormNotFound = errors.New("kazaam x-form not found in registry")

func (r Registry) Get(id string) (Transformation, error) {
	const semLogContext = "kzxform-registry::get"

	if id == "" {
		err := errors.New("transformation require an id")
		log.Error().Err(err).Msg(semLogContext)
		return Transformation{}, err
	}

	t, ok := r[id]
	if !ok {
		log.Warn().Err(XFormNotFound).Str("id", id).Msg(semLogContext)
		return Transformation{}, XFormNotFound
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

func ApplyKazaamTransformation(transformationJson []byte, data []byte) ([]byte, error) {
	const semLogContext = "transform-registry::apply-kazaan-transformation"

	transformationConfig := Config{}
	err := yaml.Unmarshal(transformationJson, &transformationConfig)
	if err != nil {
		return nil, err
	}

	rule, err := transformationConfig.ToJSONRule()
	if err != nil {
		return nil, err
	}

	k, err := kazaam.New(rule, kc)

	if transformationConfig.Verbose {
		log.Trace().Str("id", transformationConfig.Id).Str("input", string(data)).Msg(semLogContext)
	}
	dataOut, err := k.TransformJSONString(string(data))
	if transformationConfig.Verbose {
		if err != nil {
			log.Error().Err(err).Str("id", transformationConfig.Id).Msg(semLogContext)
		} else {
			log.Trace().Str("id", transformationConfig.Id).Str("output", string(data)).Msg(semLogContext)
		}
	}

	return dataOut, err
}
