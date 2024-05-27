package operators

import (
	"encoding/json"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

type ShiftArrayParams struct {
	sourceRef           JsonReference
	destRef             JsonReference
	inPlace             bool
	itemRulesSerialized string
}

func getShiftParamsFromSpec(spec *transform.Config) (ShiftArrayParams, error) {
	const semLogContext = "kazaam-shift-array::get-params-from-specs"
	var err error

	params := ShiftArrayParams{}

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

	itemRules, err := getArrayParam(spec, SpecParamSubRules, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	log.Debug().Str(SpecParamTargetReference, params.sourceRef.Path).Bool("in-place", params.inPlace).Msg(semLogContext)

	if itemRules != nil {
		b, err := json.Marshal(itemRules)
		if err != nil {
			return params, err
		}

		params.itemRulesSerialized = string(b)
		log.Debug().Str(SpecParamSubRules, params.itemRulesSerialized).Msg(semLogContext)
	}

	return params, nil
}

func ShiftArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-shift-array-items::execute"
		var err error

		params, err := getShiftParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		itemKTransformation, err := kazaam.New(params.itemRulesSerialized, kc)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		sourceArray, err := getJsonArray(data, params.sourceRef)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		// copiedData := make([]byte, len(data))
		// _ = copy(copiedData, data)
		//var modifiedArray = []byte(`{"val": []}`)
		arrayItemNdx := 0
		result := []byte(`{}`)

		var loopErr error
		_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {

			if loopErr != nil {
				log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
				return
			}

			itemTransformed, err := itemKTransformation.TransformJSONStringToString(string(value))
			if err != nil {
				// Note: how to signal back an error?
				log.Error().Err(err).Msg(semLogContext)
				loopErr = err
				return
			}

			log.Debug().Str("item-transformed", itemTransformed).Msg(semLogContext)
			result, err = jsonparser.Set(result, []byte(itemTransformed), OperatorsTempReusltPropertyName, "[+]")
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

		if arrayItemNdx > 0 {
			val, dt, _, err := jsonparser.Get(result, OperatorsTempReusltPropertyName)
			if err != nil {
				// Note: how to signal back an error?
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
			log.Info().Interface("data-type", dt).Msg(semLogContext)

			data, err = jsonparser.Set(data, val, params.destRef.Keys...)
			if err != nil {
				// Note: how to signal back an error?
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		} else {
			data, err = jsonparser.Set(data, []byte(`[]`), params.destRef.Keys...)
			if err != nil {
				// Note: how to signal back an error?
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		}

		return data, err
	}

}
