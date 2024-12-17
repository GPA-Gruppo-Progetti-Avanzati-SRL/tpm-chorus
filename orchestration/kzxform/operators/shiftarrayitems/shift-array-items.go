package shiftarrayitems

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	OperatorShiftArrayItems = "shift-array-items"
	OperatorSemLogContext   = "shift-array-items"
)

func ShiftArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"
		var err error

		params, err := getParamsFromSpec(spec)
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
			sourceArray, err := operators.GetJsonArray(data, params.sourceRef)
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

				itemTransformed := string(value)

				accepted, err := isAccepted(value, params.criteria)
				if accepted {
					itemTransformed, err = itemKTransformation.TransformJSONStringToString(itemTransformed)
				}
				if err != nil {
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

func processShiftWIthINdxSpecifier(data []byte, params OperatorParams, kXForm *kazaam.Kazaam) ([]byte, error) {
	const semLogContext = "kazaam-shift-array-items::process-i-wildcard"

	rootRef := operators.JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     params.sourceRef.Path[:strings.Index(params.sourceRef.Path, "[i]")],
		Keys:                     params.sourceRef.Keys[:params.sourceRef.WithArrayISpecifierIndex],
	}

	rootArray, err := operators.GetJsonArray(data, rootRef)
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

		nestedRef := operators.JsonReference{
			WithArrayISpecifierIndex: -1,
			Path:                     strings.ReplaceAll(params.sourceRef.Path[strings.Index(params.sourceRef.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", loopIndex)),
			// Keys:                     make([]string, len(sourceRef.Keys)),
		}

		nestedRef.Keys = append(nestedRef.Keys, params.sourceRef.Keys[params.sourceRef.WithArrayISpecifierIndex:]...)
		// copy(nestedRef.Keys, sourceRef.Keys)
		nestedRef.Keys[0] = fmt.Sprintf("[%d]", loopIndex)
		resultArray, err := processShiftArray(rootArray, nestedRef, kXForm, params.criteria)
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

func processShiftArray(data []byte, sourceRef operators.JsonReference, itemKTransformation *kazaam.Kazaam, criteria []operators.Criterion) ([]byte, error) {
	const semLogContext = "kazaam-filter-array-items::process-array"

	sourceArray, err := operators.GetJsonArray(data, sourceRef)
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

		itemTransformed := string(value)
		accepted, err := isAccepted(value, criteria)
		if accepted {
			itemTransformed, err = itemKTransformation.TransformJSONStringToString(itemTransformed)
		}
		if err != nil {
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

func isAccepted(value []byte, obj []operators.Criterion) (bool, error) {

	// Always accepted if not specified
	if len(obj) == 0 {
		return true, nil
	}

	for _, criterion := range obj {
		attributeValue, dataType, _, err := jsonparser.Get(value, criterion.AttributeName.Keys...)
		if err != nil {
			return false, err
		}

		if dataType == jsonparser.NotExist {
			continue
		}

		attrValue := string(attributeValue)
		if attrValue == criterion.Term {
			return true, nil
		}
	}

	return false, nil
}
