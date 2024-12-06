package operators

import (
	"encoding/json"
	"fmt"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
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

		if params.sourceRef.WithArrayISpecifierIndex < 0 {
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
		} else {
			data, err = processShiftWIthINdxSpecifier(data, params, itemKTransformation)
		}

		return data, err
	}

}

func processShiftWIthINdxSpecifier(data []byte, params ShiftArrayParams, kXForm *kazaam.Kazaam) ([]byte, error) {
	const semLogContext = "kazaam-shift-array-items::process-i-wildcard"

	rootRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     params.sourceRef.Path[:strings.Index(params.sourceRef.Path, "[i]")],
		Keys:                     params.sourceRef.Keys[:params.sourceRef.WithArrayISpecifierIndex],
	}

	rootArray, err := getJsonArray(data, rootRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	// clone the data... in place process has some glitches.
	outData := make([]byte, len(data))
	copy(outData, data)

	var loopErr error
	var loopIndex int
	_, err = jsonparser.ArrayEach(rootArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		nestedRef := JsonReference{
			WithArrayISpecifierIndex: -1,
			Path:                     strings.ReplaceAll(params.sourceRef.Path[strings.Index(params.sourceRef.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", loopIndex)),
			// Keys:                     make([]string, len(sourceRef.Keys)),
		}

		nestedRef.Keys = append(nestedRef.Keys, params.sourceRef.Keys[params.sourceRef.WithArrayISpecifierIndex:]...)
		// copy(nestedRef.Keys, sourceRef.Keys)
		nestedRef.Keys[0] = fmt.Sprintf("[%d]", loopIndex)
		resultArray, err := processShiftArray(rootArray, nestedRef, kXForm)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		var nestedTargetRefKeys []string
		if params.destRef.WithArrayISpecifierIndex >= 0 {
			nestedTargetRefKeys = append(nestedTargetRefKeys, params.destRef.Keys...)
			nestedTargetRefKeys[params.destRef.WithArrayISpecifierIndex] = fmt.Sprintf("[%d]", loopIndex)
		} else {
			nestedTargetRefKeys = append(nestedTargetRefKeys, params.destRef.Keys...)
		}
		outData, err = jsonparser.Set(outData, resultArray, nestedTargetRefKeys...)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		loopIndex++
	})

	return outData, loopErr
}

func processShiftArray(data []byte, sourceRef JsonReference, itemKTransformation *kazaam.Kazaam) ([]byte, error) {
	const semLogContext = "kazaam-filter-array-items::process-array"

	sourceArray, err := getJsonArray(data, sourceRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	// Variables to build new array
	arrayItemNdx := 0
	resultArray := []byte(`{}`)
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

		resultArray, err = jsonparser.Set(resultArray, []byte(itemTransformed), OperatorsTempReusltPropertyName, "[+]")
		if err != nil {
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
		var dt jsonparser.ValueType
		resultArray, dt, _, err = jsonparser.Get(resultArray, OperatorsTempReusltPropertyName)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Info().Interface("data-type", dt).Msg(semLogContext)
	} else {
		resultArray = []byte(`[]`)
	}

	return resultArray, nil
}
