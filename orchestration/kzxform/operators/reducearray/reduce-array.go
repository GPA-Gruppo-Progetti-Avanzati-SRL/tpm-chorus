package reducearray

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"

	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	OperatorReduceArray   = "reduce-array"
	OperatorSemLogContext = OperatorReduceArray
)

func ReduceArray(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"
		var err error

		params, err := getParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		log.Debug().Str(SpecParamSourceReference, params.sourceRef.Path).Str(SpecParamPropertyNameRef, params.propertyNameRef.Path).Str(SpecParamPropertyValueRef, params.propertyValueRef.Path).Bool("in-place", params.inPlace).Msg(semLogContext)

		targetArray, err := operators.GetJsonArray(data, params.sourceRef)
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

			propertyName, err := operators.GetJsonString(value, params.propertyNameRef.Path, false)
			if err != nil {
				// Note: how to signal back an error?
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			if propertyName == "" {
				return
			}

			propertyValueValue, dataType, err := operators.GetJsonValue(value, params.propertyValueRef.Path)
			if err != nil {
				// Note: how to signal back an error?
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			if propertyValueValue == nil {
				return
			}
			if dataType == jsonparser.String {
				quote := []byte("\"")
				propertyValueValue = append(quote, append(propertyValueValue, quote...)...)
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
