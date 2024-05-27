package operators

import (
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

type ReduceArrayParams struct {
	sourceRef        JsonReference
	destRef          JsonReference
	inPlace          bool
	propertyNameRef  JsonReference
	propertyValueRef JsonReference
}

func getReduceParamsFromSpec(spec *transform.Config) (ReduceArrayParams, error) {
	const semLogContext = "kazaam-reduce-array::get-params-from-specs"
	var err error

	params := ReduceArrayParams{}
	params.sourceRef, err = getJsonReferenceParam(spec, SpecParamSourceReference, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.destRef, err = getJsonReferenceParam(spec, SpecParamTargetReference, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.inPlace = false
	if params.destRef.IsZero() || params.sourceRef.Path == params.destRef.Path {
		params.destRef = params.sourceRef
		params.inPlace = true
	}

	params.propertyNameRef, err = getJsonReferenceParam(spec, SpecParamPropertyNameRef, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.propertyValueRef, err = getJsonReferenceParam(spec, SpecParamPropertyValueRef, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	log.Debug().Str(SpecParamSourceReference, params.sourceRef.Path).Str(SpecParamPropertyNameRef, params.propertyNameRef.Path).Str(SpecParamPropertyValueRef, params.propertyValueRef.Path).Bool("in-place", params.inPlace).Msg(semLogContext)
	return params, nil
}

func ReduceArray(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-reduce-array::execute"
		var err error

		params, err := getReduceParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		log.Debug().Str(SpecParamSourceReference, params.sourceRef.Path).Str(SpecParamPropertyNameRef, params.propertyNameRef.Path).Str(SpecParamPropertyValueRef, params.propertyValueRef.Path).Bool("in-place", params.inPlace).Msg(semLogContext)

		targetArray, err := getJsonArray(data, params.sourceRef)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		arrayItemNdx := 0
		result := []byte(`{}`)

		var loopErr error
		_, err = jsonparser.ArrayEach(targetArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {

			if loopErr != nil {
				log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
				return
			}

			propertyName, err := getJsonString(value, params.propertyNameRef.Path, false)
			if err != nil {
				// Note: how to signal back an error?
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			if propertyName == "" {
				return
			}

			propertyValueValue, err := getJsonValue(value, params.propertyValueRef.Path)
			if err != nil {
				// Note: how to signal back an error?
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			if propertyValueValue == nil {
				return
			}

			result, err = jsonparser.Set(result, propertyValueValue, propertyName)
			if err != nil {
				// Note: how to signal back an error?
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			arrayItemNdx++
		})

		if loopErr != nil {
			return nil, loopErr
		}

		data, err = jsonparser.Set(data, result, params.destRef.Keys...)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		return data, err
	}

}
