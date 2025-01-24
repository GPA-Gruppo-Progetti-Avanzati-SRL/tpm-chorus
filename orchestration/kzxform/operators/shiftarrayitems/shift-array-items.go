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
	OperatorSemLogContext   = "kz-shift-array-items"
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
			data, err = process(data, params.sourceRef, itemKTransformation, params)
		} else {
			data, err = processWithIotaSpecifier(data, params, itemKTransformation)
		}

		return data, err
	}

}

/*
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
*/

func process(data []byte, sourceRef operators.JsonReference, itemKTransformation *kazaam.Kazaam, params OperatorParams) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process"
	result, err := processArray(data, sourceRef, itemKTransformation, params)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	data, err = jsonparser.Set(data, result, params.destRef.Keys...)
	if err != nil {
		// Note: how to signal back an error?
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return data, err
}

func processWithIotaSpecifier(data []byte, params OperatorParams, kXForm *kazaam.Kazaam) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process-with-i-specifier"

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
		resultArray, err := processArray(rootArray, nestedRef, kXForm, params)
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
