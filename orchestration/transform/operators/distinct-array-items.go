package operators

import (
	"bytes"
	"errors"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

type DistinctArrayItemsParams struct {
	sourceRef JsonReference
	destRef   JsonReference
	On        JsonReference
}

func getDistinctArrayItemsParamsFromSpec(spec *transform.Config) (DistinctArrayItemsParams, error) {
	const semLogContext = "kazaam-distinct-array-items::get-params-from-specs"
	var err error

	params := DistinctArrayItemsParams{}

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

	params.On, err = getJsonReferenceParam(spec, SpecParamDistinctOn, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	return params, nil
}

func DistinctArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-filter-array-items::execute"

		params, err := getDistinctArrayItemsParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		// clone the data... in place process has some glitches.
		outData := make([]byte, len(data))
		copy(outData, data)

		sourceArray, err := getJsonArray(data, params.sourceRef)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		distinctMap := make(map[string][]byte)

		var loopErr error
		_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {

			if loopErr != nil {
				log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
				return
			}

			mapKey, err := getDistinctMapKey(value, params.On)
			if err != nil {
				// Note: how to signal back an error?
				log.Error().Err(err).Msg(semLogContext)
				loopErr = err
				return
			}

			distinctMap[mapKey] = value
		})

		if loopErr != nil {
			return nil, loopErr
		}

		var sb bytes.Buffer
		sb.WriteString("[")
		if len(distinctMap) > 0 {
			i := 0
			for _, v := range distinctMap {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.Write(v)
				i++
			}
		}
		sb.WriteString("]")
		outData, err = jsonparser.Set(outData, sb.Bytes(), params.destRef.Keys...)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		return outData, err
	}

}

func getDistinctMapKey(value []byte, on JsonReference) (string, error) {

	attributeValue, dataType, _, err := jsonparser.Get(value, on.Keys...)
	if err != nil {
		return "", err
	}

	if dataType == jsonparser.NotExist {
		return "", errors.New("attribut doesn't exists")
	}

	attrValue := string(attributeValue)
	return attrValue, nil
}
