package shiftarrayitems

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/rs/zerolog/log"
)

func processArray(data []byte, sourceRef operators.JsonReference, itemKTransformation *kazaam.Kazaam, params OperatorParams) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process-array"

	sourceArray, err := operators.GetJsonArray(data, sourceRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	lenOfSourceArray, err := operators.LenOfArray(sourceArray, operators.JsonReference{})
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	log.Info().Int("len", lenOfSourceArray).Msg(semLogContext)

	// copiedData := make([]byte, len(data))
	// _ = copy(copiedData, data)
	//var modifiedArray = []byte(`{"val": []}`)
	arrayItemNdx := 0
	resultArray := []byte(`{}`)
	resultLen := 0
	var loopErr error
	_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {

		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		itemTransformed := string(value)
		if dataType == jsonparser.String {
			itemTransformed = fmt.Sprintf(`"%s"`, itemTransformed)
		}

		accepted := true
		if len(params.criteria) > 0 {
			accepted, err = params.criteria.IsAccepted(value, map[string]interface{}{operators.CriterionSystemVariableKazaamArrayLen: lenOfSourceArray, operators.CriterionSystemVariableKazaamArrayLoopIndex: arrayItemNdx}) // isAccepted(value, params.criteria)
		}
		if accepted {
			itemTransformed, err = itemKTransformation.TransformJSONStringToString(itemTransformed)
		}
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		if accepted || !params.filterItems {
			resultArray, err = jsonparser.Set(resultArray, []byte(itemTransformed), OperatorsTempResultPropertyName, "[+]")
			if err != nil {
				// Note: how to signal back an error?
				loopErr = err
				log.Error().Err(err).Msg(semLogContext)
				return
			}
			resultLen++
		}

		arrayItemNdx++
	})

	if loopErr != nil {
		return nil, loopErr
	}

	var rcItem []byte
	switch resultLen {
	case 0:
		if params.flatten {
			rcItem = []byte(`{}`)
		} else {
			rcItem = []byte(`[]`)
		}

	case 1:
		if params.flatten {
			var item0 []byte
			var dt jsonparser.ValueType
			item0, dt, _, err = jsonparser.Get(resultArray, OperatorsTempResultPropertyName, "[0]")
			log.Trace().Str("dt", dt.String()).Msg(semLogContext)
			rcItem = []byte(`{}`)
			err = jsonparser.ObjectEach(item0, func(key, value []byte, dataType jsonparser.ValueType, offset int) error {

				if dataType == jsonparser.String {
					value = []byte(fmt.Sprintf(`"%s"`, string(value)))
				}
				rcItem, err = jsonparser.Set(rcItem, value, string(key))
				return nil
			})
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		} else {
			var dt jsonparser.ValueType
			rcItem, dt, _, err = jsonparser.Get(resultArray, OperatorsTempResultPropertyName)
			if err != nil {
				// Note: how to signal back an error?
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
			log.Info().Interface("data-type", dt).Msg(semLogContext)
		}
	default:
		var dt jsonparser.ValueType
		rcItem, dt, _, err = jsonparser.Get(resultArray, OperatorsTempResultPropertyName)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Info().Interface("data-type", dt).Msg(semLogContext)
	}

	/*
		if resultLen > 0 {
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
	*/

	return rcItem, err
}
