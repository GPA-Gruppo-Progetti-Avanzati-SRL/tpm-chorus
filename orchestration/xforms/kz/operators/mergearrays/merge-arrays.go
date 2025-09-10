package mergearrays

import (
	"errors"
	"fmt"
	"strings"

	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	OperatorMergeArrays                 = "merge-arrays"
	OperatorSemLogContext               = OperatorMergeArrays
	OperatorMergeArraysTempPropertyName = "merge-arrays-tmp"
)

func MergeArrays(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"

		params, err := getParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		if len(params.mapping) == 0 {
			return data, nil
		}

		// clone the data... in place process has some glitches.
		outData := make([]byte, len(data))
		copy(outData, data)

		for _, m := range params.mapping {
			if m.src.WithArrayISpecifierIndex < 0 {
				err = errors.New("invalid specs: the source requires a sub-array reference with iota specifier")
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			outData, err = process(outData, m.dst, data, m.src)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		}

		return outData, err
	}
}

func process(outData []byte, dst operators2.JsonReference, data []byte, src operators2.JsonReference) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process"

	rootRef := operators2.JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     src.Path[:strings.Index(src.Path, "[i]")],
		Keys:                     src.Keys[:src.WithArrayISpecifierIndex],
	}

	rootArray, err := operators2.GetJsonArray(data, rootRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	var loopErr error
	var loopIndex int
	mergedArray := []byte(`{}`)
	mergedArraySize := 0
	_, err = jsonparser.ArrayEach(rootArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		// Questo mapping crea riferimenti del tipo [0].prop1.prop2, [1].prop1.prop2.... toglie la parte a monte per eseguire il mapping
		// con la variabile rootArray
		nestedRef := operators2.JsonReference{
			WithArrayISpecifierIndex: -1,
			Path:                     strings.ReplaceAll(src.Path[strings.Index(src.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", loopIndex)),
			// Keys:                     make([]string, len(sourceRef.Keys)),
		}
		nestedRef.Keys = append(nestedRef.Keys, src.Keys[src.WithArrayISpecifierIndex:]...)
		nestedRef.Keys[0] = fmt.Sprintf("[%d]", loopIndex)

		// var nestedLen = 0
		numMergedItems := 0
		mergedArray, numMergedItems, err = mergeArray(mergedArray, rootArray, nestedRef)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		mergedArraySize += numMergedItems
		loopIndex++
	})

	var resultArray []byte
	if mergedArraySize > 0 {
		var dt jsonparser.ValueType
		resultArray, dt, _, err = jsonparser.Get(mergedArray, OperatorMergeArraysTempPropertyName)
		if err != nil {
			// Note: how to signal back an error?
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Info().Interface("data-type", dt).Msg(semLogContext)
	} else {
		resultArray = []byte(`[]`)
	}

	var nestedTargetRefKeys []string
	nestedTargetRefKeys = append(nestedTargetRefKeys, dst.Keys...)
	outData, err = jsonparser.Set(outData, resultArray, nestedTargetRefKeys...)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return outData, nil
}

func mergeArray(outArray []byte, data []byte, jsonRef operators2.JsonReference) ([]byte, int, error) {
	const semLogContext = OperatorSemLogContext + "::merge-array"

	nestedArray, err := operators2.GetJsonArray(data, jsonRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, -1, err
	}

	nestedLoopIndex := 0
	var loopErr error
	_, err = jsonparser.ArrayEach(nestedArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		nestedLoopIndex++

		if loopErr != nil {
			return
		}

		if dataType == jsonparser.String {
			newValue := make([]byte, len(value)+2)
			newValue[0] = '"'
			newValue[len(newValue)-1] = '"'
			copy(newValue[1:], value)
			value = newValue
		}

		outArray, err = jsonparser.Set(outArray, value, OperatorMergeArraysTempPropertyName, "[+]")
		if err != nil {
			loopErr = err
			log.Error().Err(err).Msg(semLogContext)
			return
		}
	})

	return outArray, nestedLoopIndex, loopErr
}
