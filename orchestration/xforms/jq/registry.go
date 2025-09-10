package jq

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms"
	"github.com/itchyny/gojq"
	"github.com/qntfy/kazaam"
	"github.com/rs/zerolog/log"
)

var kc kazaam.Config

type Transformation struct {
	Cfg    xforms.TransformReference
	JQCode *gojq.Code
}

type Registry map[string]Transformation

var registry Registry

func InitializeJQRegistry() error {
	registry = make(map[string]Transformation)
	return nil
}

func GetRegistry() Registry {

	const semLogContext = "jq-xform-registry::get-registry"

	if registry == nil {
		err := InitializeJQRegistry()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}
	}

	return registry
}

func (r Registry) AddTransformation(ref xforms.TransformReference) error {

	query, err := gojq.Parse(string(ref.Data))
	if err != nil {
		return err
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return err
	}

	trsf := Transformation{
		Cfg:    ref,
		JQCode: code,
	}

	err = r.Add(trsf)
	if err != nil {
		return err
	}

	return nil
}

func (r Registry) Add(xform Transformation) error {

	const semLogContext = "jq-xform-registry::add-xform"
	if xform.Cfg.Id == "" {
		err := errors.New("transformation require an id")
		return err
	}

	if _, ok := r[xform.Cfg.Id]; ok {
		err := fmt.Errorf("transformation id must be unique (conflicting id: %s)", xform.Cfg.Id)
		log.Warn().Err(err).Msg(semLogContext)
		return nil
	}

	r[xform.Cfg.Id] = xform

	return nil
}

var XFormNotFound = errors.New("jq x-form not found in registry")

func (r Registry) Get(id string) (Transformation, error) {
	const semLogContext = "jq-xform-registry::get-xform"

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
	const semLogContext = "jq-xform-registry::transform"

	log.Debug().Str("id", id).Msg(semLogContext)
	t, err := r.Get(id)
	if err != nil {
		return nil, err
	}

	var indataJsonAsObject any
	err = json.Unmarshal(data, &indataJsonAsObject)
	if err != nil {
		return nil, err
	}

	m, err := applyJQCompiledTransformation(t.JQCode, indataJsonAsObject)
	if err != nil {
		return nil, err
	}

	dataOut, err := json.Marshal(m[0])
	if err != nil {
		return nil, err
	}

	return dataOut, nil
}

func ApplyJQTransformationToJson(xformText []byte, data []byte) ([]byte, error) {
	var indataJsonAsObject any
	err := json.Unmarshal(data, &indataJsonAsObject)
	if err != nil {
		return nil, err
	}

	m, err := ApplyJQTransformation(xformText, indataJsonAsObject)
	if err != nil {
		return nil, err
	}

	dataOut, err := json.Marshal(m[0])
	if err != nil {
		return nil, err
	}

	return dataOut, nil
}

func ApplyJQTransformation(xformText []byte, data any) ([]any, error) {
	const semLogContext = "jq-xform::apply-jq-transformation"

	query, err := gojq.Parse(string(xformText))
	if err != nil {
		return nil, err
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return nil, err
	}

	return applyJQCompiledTransformation(code, data)
}

func applyJQCompiledTransformation(code *gojq.Code, data any) ([]any, error) {
	const semLogContext = "jq-xform::apply-jq-compiled-transformation"

	iter := code.Run(data)
	var withErrors bool
	var dataOut []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if err, ok := v.(error); ok {
			var hErr *gojq.HaltError
			if errors.As(err, &hErr) {
				return nil, err
			}

			log.Error().Str("err", err.Error()).Msg(semLogContext)
			withErrors = true
			continue
		} else {
			dataOut = append(dataOut, v)
		}
	}

	if withErrors {
		return dataOut, errors.New("errors in running jq transformation")
	}

	return dataOut, nil
}
