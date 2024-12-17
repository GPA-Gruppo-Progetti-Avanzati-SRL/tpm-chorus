package filterarrayitems

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
	OperatorFilterArrayItems = "filter-array-items"
	OperatorSemLogContext    = OperatorFilterArrayItems
)

func FilterArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"

		params, err := getFilterParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		/*
			sourceRef, err := getJsonReferenceParam(spec, SpecParamSourceReference, true)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			destRef, err := getJsonReferenceParam(spec, SpecParamTargetReference, false)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			inPlace := false
			if destRef.IsZero() || sourceRef.Path == destRef.Path {
				destRef = sourceRef
				inPlace = true
			}

			filters, err := getArrayParam(spec, SpecParamCriteria, true)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			filtersObj, err := getFilterConfigsFromSpec(filters)
			log.Debug().Interface(SpecParamSourceReference, sourceRef).Interface(SpecParamTargetReference, destRef).Interface(SpecParamCriteria, filtersObj).Bool("in-place", inPlace).Msg(semLogContext)
		*/

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
			//itemKeys := make([]string, len(targetArrayKeys)+1)
			//_ = copy(itemKeys, targetArrayKeys)
			//itemKeys[len(itemKeys)-1] = fmt.Sprintf("[+]")

			// transformedData := jsonparser.Delete(data, targetArrayKeys...)
			filteredArray := []byte(`{}`)

			var loopErr error
			_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {

				if loopErr != nil {
					log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
					return
				}

				accepted, err := isAccepted(value, params.criteria)
				if err != nil {
					// Note: how to signal back an error?
					log.Error().Err(err).Msg(semLogContext)
					loopErr = err
					return
				}

				if accepted {
					filteredArray, err = jsonparser.Set(filteredArray, value, OperatorsTempReusltPropertyName, "[+]")
					// transformedData, err = jsonparser.Set(transformedData, []byte(value), itemKeys...)
					if err != nil {
						// Note: how to signal back an error?
						loopErr = err
						log.Error().Err(err).Msg(semLogContext)
						return
					}

					arrayItemNdx++
				}
			})

			if loopErr != nil {
				return nil, loopErr
			}

			if arrayItemNdx > 0 {
				val, dt, _, err := jsonparser.Get(filteredArray, OperatorsTempReusltPropertyName)
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
			data, err = processFilterWIthINdxSpecifier(data, params)
		}

		return data, err
	}

}

func isAccepted(value []byte, obj []operators.Criterion) (bool, error) {

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

func processFilterWIthINdxSpecifier(data []byte, params FilterArrayParams) ([]byte, error) {
	const semLogContext = "kazaam-filter-array-items::process-i-wildcard"

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
		filteredArray, err := processArray(rootArray, nestedRef, params.criteria)
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
		outData, err = jsonparser.Set(outData, filteredArray, nestedTargetRefKeys...)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		loopIndex++
	})

	return outData, loopErr
}

func processArray(data []byte, sourceRef operators.JsonReference, criteria []operators.Criterion) ([]byte, error) {
	const semLogContext = "kazaam-filter-array-items::process-array"

	sourceArray, err := operators.GetJsonArray(data, sourceRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	// Variables to build new array
	arrayItemNdx := 0
	filteredArray := []byte(`{}`)
	var loopErr error
	_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		accepted, err := isAccepted(value, criteria)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		if accepted {
			filteredArray, err = jsonparser.Set(filteredArray, value, OperatorsTempReusltPropertyName, "[+]")
			if err != nil {
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			arrayItemNdx++
		}
	})

	if loopErr != nil {
		return nil, loopErr
	}

	if arrayItemNdx > 0 {
		var dt jsonparser.ValueType
		filteredArray, dt, _, err = jsonparser.Get(filteredArray, OperatorsTempReusltPropertyName)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Info().Interface("data-type", dt).Msg(semLogContext)
	} else {
		filteredArray = []byte(`[]`)
	}

	return filteredArray, nil
}
