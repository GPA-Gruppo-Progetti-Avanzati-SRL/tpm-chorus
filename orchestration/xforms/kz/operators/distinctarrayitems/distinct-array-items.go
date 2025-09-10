package distinctarrayitems

import (
	"bytes"
	"errors"

	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	OperatorDistinctArrayItems = "distinct-items"
	OperatorSemLogContext      = OperatorDistinctArrayItems
)

func DistinctArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"

		params, err := getDistinctArrayItemsParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		// clone the data... in place process has some glitches.
		outData := make([]byte, len(data))
		copy(outData, data)

		sourceArray, err := operators2.GetJsonArray(data, params.sourceRef)
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

func getDistinctMapKey(value []byte, on operators2.JsonReference) (string, error) {

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
