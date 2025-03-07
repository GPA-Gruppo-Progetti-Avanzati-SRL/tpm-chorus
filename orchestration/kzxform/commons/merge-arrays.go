package commons

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/qntfy/jsonparser"
	"github.com/rs/zerolog/log"
)

func MergeArrays(a1, a2 any) interface{} {
	const semLogContext = "operators-funcs::merge-arrays"

	e1, err1 := mergeArraysValidateArg(a1)
	e2, err2 := mergeArraysValidateArg(a2)
	err := util.CoalesceError(err1, err2)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil
	}

	if e1.Dt == jsonparser.NotExist || e1.Dt == jsonparser.Null {
		log.Trace().Msg(semLogContext + " - second param is null, returning second")
		return e2
	} else if e2.Dt == jsonparser.NotExist || e2.Dt == jsonparser.Null {
		log.Trace().Msg(semLogContext + " - second param is null, returning first")
		return e1
	}

	mergedArray := []byte(`{}`)
	numItemsMerged := 0
	numItems := 0
	mergedArray, numItems, err = mergeArray(mergedArray, e1.Val)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return NullVar
	}
	numItemsMerged += numItems
	log.Trace().Int("source-num-items", numItems).Int("merged-num-items", numItemsMerged).Msg(semLogContext + " - merged first array")

	mergedArray, numItems, err = mergeArray(mergedArray, e2.Val)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return NullVar
	}
	numItemsMerged += numItems
	log.Trace().Int("source-num-items", numItems).Int("merged-num-items", numItemsMerged).Msg(semLogContext + " - merged second array")

	var resultArray []byte
	if numItemsMerged > 0 {
		var dt jsonparser.ValueType
		resultArray, dt, _, err = jsonparser.Get(mergedArray, "__func_merge_array__")
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return NullVar
		}
		log.Info().Interface("data-type", dt).Msg(semLogContext)
	} else {
		resultArray = []byte(`[]`)
	}

	return ExpressionVariable{Dt: jsonparser.Array, Val: resultArray}
}

func mergeArray(outArray []byte, arr []byte) ([]byte, int, error) {
	const semLogContext = "operators-funcs::merge-array"
	var err error

	nestedLoopIndex := 0
	var loopErr error
	_, err = jsonparser.ArrayEach(arr, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
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

		outArray, err = jsonparser.Set(outArray, value, "__func_merge_array__", "[+]")
		if err != nil {
			loopErr = err
			log.Error().Err(err).Msg(semLogContext)
			return
		}
	})

	return outArray, nestedLoopIndex, loopErr
}

func mergeArraysValidateArg(a any) (ExpressionVariable, error) {
	const semLogContext = "operators-funcs::merge-arrays-validate-arg"
	var err error

	e := ExpressionVariable{Dt: jsonparser.Null}
	switch param := a.(type) {
	case ExpressionVariable:
		switch param.Dt {
		case jsonparser.Array:
			e = param
		case jsonparser.Null:
			fallthrough
		case jsonparser.NotExist:
			log.Info().Str("var-dt", fmt.Sprint(param.Dt)).Msg(semLogContext)
			e = param
		default:
			err = errors.New("function not applicable")
			log.Warn().Err(err).Str("var-dt", fmt.Sprint(param.Dt)).Msg(semLogContext)
		}

	default:
		err = errors.New("unsupported type")
		log.Warn().Err(err).Str("data-type", fmt.Sprintf("%T", a)).Msg(semLogContext)
	}

	return e, err
}
