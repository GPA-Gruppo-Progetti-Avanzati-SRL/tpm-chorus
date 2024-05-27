package operators

import (
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

type FilterArrayParams struct {
	sourceRef JsonReference
	destRef   JsonReference
	inPlace   bool
	criteria  []filterCfg
}

func getFilterParamsFromSpec(spec *transform.Config) (FilterArrayParams, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"
	var err error

	params := FilterArrayParams{}

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

	filters, err := getArrayParam(spec, SpecParamCriteria, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.criteria, err = getFilterConfigsFromSpec(filters)
	log.Debug().Interface(SpecParamSourceReference, params.sourceRef).Interface(SpecParamTargetReference, params.destRef).Interface(SpecParamCriteria, params.criteria).Bool("in-place", params.inPlace).Msg(semLogContext)
	return params, nil
}

type filterCfg struct {
	attributeName JsonReference
	operator      string
	term          string
}

func getFilterConfigFromSpec(c interface{}) (filterCfg, error) {
	var err error
	var criterion filterCfg
	criterion.attributeName, err = getJsonReferenceParamFromMap(c, SpecParamCriterionAttributeReference, true)
	if err != nil {
		return criterion, err
	}

	criterion.operator = "eq"

	criterion.term, err = getStringParamFromMap(c, SpecParamCriterionTerm, true)
	if err != nil {
		return criterion, err
	}

	return criterion, nil
}

func getFilterConfigsFromSpec(c []interface{}) ([]filterCfg, error) {
	filtersObj := make([]filterCfg, 0)
	for _, f := range c {
		crit, err := getFilterConfigFromSpec(f)
		if err != nil {
			return nil, err
		}

		filtersObj = append(filtersObj, crit)
	}

	return filtersObj, nil
}

func FilterArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-filter-array-items::execute"
		var err error

		params, err := getFilterParamsFromSpec(spec)

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
		sourceArray, err := getJsonArray(data, params.sourceRef)
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
	}

}

func isAccepted(value []byte, obj []filterCfg) (bool, error) {

	for _, criterion := range obj {
		attributeValue, dataType, _, err := jsonparser.Get(value, criterion.attributeName.Keys...)
		if err != nil {
			return false, err
		}

		if dataType == jsonparser.NotExist {
			continue
		}

		attrValue := string(attributeValue)
		if attrValue == criterion.term {
			return true, nil
		}
	}

	return false, nil
}
