package reducearray

import (
	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	SpecParamSourceReference  = "source-ref"
	SpecParamTargetReference  = "target-ref"
	SpecParamPropertyNameRef  = "name-ref"
	SpecParamPropertyValueRef = "value-ref"
)

type ReduceArrayParams struct {
	sourceRef        operators2.JsonReference
	destRef          operators2.JsonReference
	inPlace          bool
	propertyNameRef  operators2.JsonReference
	propertyValueRef operators2.JsonReference
}

func getParamsFromSpec(spec *transform.Config) (ReduceArrayParams, error) {
	const semLogContext = "kazaam-reduce-array::get-params-from-specs"
	var err error

	params := ReduceArrayParams{}
	params.sourceRef, err = operators2.GetJsonReferenceParam(spec, SpecParamSourceReference, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.destRef, err = operators2.GetJsonReferenceParam(spec, SpecParamTargetReference, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.inPlace = false
	if params.destRef.IsZero() || params.sourceRef.Path == params.destRef.Path {
		params.destRef = params.sourceRef
		params.inPlace = true
	}

	params.propertyNameRef, err = operators2.GetJsonReferenceParam(spec, SpecParamPropertyNameRef, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.propertyValueRef, err = operators2.GetJsonReferenceParam(spec, SpecParamPropertyValueRef, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	log.Debug().
		Str(SpecParamSourceReference, params.sourceRef.Path).
		Str(SpecParamPropertyNameRef, params.propertyNameRef.Path).
		Str(SpecParamPropertyValueRef, params.propertyValueRef.Path).
		Bool("in-place", params.inPlace).
		Msg(semLogContext)
	return params, nil
}
