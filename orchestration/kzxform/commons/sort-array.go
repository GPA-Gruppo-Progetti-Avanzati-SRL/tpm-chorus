package commons

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/qntfy/jsonparser"
	"github.com/rs/zerolog/log"
	"sort"
)

func SortArray(a1 any, propertyName string) interface{} {
	const semLogContext = "operators-funcs::sort-array"

	e1, err := mergeArraysValidateArg(a1)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil
	}

	if e1.Dt == jsonparser.NotExist || e1.Dt == jsonparser.Null {
		log.Trace().Msg(semLogContext + " - param is null")
		return NullVar
	}

	listOfItems, dataType, numItems, err := JsonArray2ListOfItems(e1.Val)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return NullVar
	}

	if numItems == 0 {
		return EmptyArrayVar
	}

	resultArray := EmptyArrayVar
	switch dataType {
	case jsonparser.String:
		resultArray = sortArrayOfStrings(listOfItems)
	case jsonparser.Number:
		err = errors.New("unsupported data type")
		log.Error().Err(err).Msg(semLogContext)

	case jsonparser.Object:
		resultArray = sortArrayOfObjects(listOfItems, propertyName)
	default:
		err = errors.New("unsupported data type")
		log.Error().Err(err).Msg(semLogContext)
	}

	return resultArray
}

func sortArrayOfStrings(in [][]byte) ExpressionVariable {
	sort.SliceStable(in, func(i, j int) bool {
		return bytes.Compare(in[i], in[j]) < 0
	})

	var sb bytes.Buffer
	sb.WriteString("[")
	for i, el := range in {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", string(el)))
	}
	sb.WriteString("]")
	return ExpressionVariable{Dt: jsonparser.Array, Val: sb.Bytes()}
}

type CmpItem struct {
	v                []byte
	originalPosition int
}

func sortArrayOfObjects(in [][]byte, propertyName string) ExpressionVariable {
	const semLogContext = "operators-funcs::sort-array-of-objects"

	if len(in) == 0 {
		return NullVar
	}

	if len(in) == 1 {
		return listOfItems2JsonArrayValue(in, false)
	}

	var listOfCmpItems []CmpItem
	itemDt := jsonparser.Unknown
	numberOfMissings := 0

	for i, el := range in {
		v, dt, _, err := jsonparser.Get(el, propertyName)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return NullVar
		}

		if dt == jsonparser.NotExist || dt == jsonparser.Null {
			log.Warn().Str("property-name", propertyName).Str("json", string(el)).Msg(semLogContext + " - property is null")
			listOfCmpItems = append(listOfCmpItems, CmpItem{v: nil, originalPosition: i})
			numberOfMissings++
		} else {
			if i == 0 {
				itemDt = dt
			} else if itemDt != dt {
				log.Error().Err(err).Msg(semLogContext)
				return listOfItems2JsonArrayValue(in, false)
			}
			listOfCmpItems = append(listOfCmpItems, CmpItem{v: v, originalPosition: i})
		}
	}

	// don't do it if property is not present....
	if numberOfMissings == len(listOfCmpItems) {
		return listOfItems2JsonArrayValue(in, false)
	}

	sort.SliceStable(listOfCmpItems, func(i, j int) bool {
		switch {
		case listOfCmpItems[i].v == nil:
			return true
		case listOfCmpItems[j].v == nil:
			return false
		default:
			return bytes.Compare(listOfCmpItems[i].v, listOfCmpItems[j].v) < 0
		}
	})

	var sorted [][]byte
	for _, el := range listOfCmpItems {
		sorted = append(sorted, in[el.originalPosition])
	}

	return listOfItems2JsonArrayValue(sorted, false)
}

func JsonArray2ListOfItems(sourceArray []byte) ([][]byte, jsonparser.ValueType, int, error) {
	const semLogContext = "operators-funcs::json-array-2-list-of-items"

	var list [][]byte
	var loopErr error
	numItems := 0
	vt := jsonparser.Unknown
	_, err := jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {

		if errParam != nil {
			log.Error().Err(errParam).Msg(semLogContext)
			loopErr = errParam
		}

		if loopErr != nil {
			return
		}

		numItems++
		list = append(list, value)
		if numItems == 1 {
			vt = dataType
		} else {
			if vt != dataType {
				vt = jsonparser.Unknown
				loopErr = errors.New("json array item data type mismatch")
				log.Error().Err(loopErr).Msg(semLogContext + " - param type mismatch")
				return
			}
		}
	})

	return list, vt, numItems, util.CoalesceError(loopErr, err)
}

func listOfItems2JsonArrayValue(in [][]byte, quoted bool) ExpressionVariable {
	var sb bytes.Buffer
	sb.WriteString("[")
	for i, el := range in {
		if i > 0 {
			sb.WriteString(",")
		}
		if quoted {
			sb.WriteString(fmt.Sprintf("\"%s\"", string(el)))
		} else {
			sb.Write(el)
		}
	}
	sb.WriteString("]")
	return ExpressionVariable{Dt: jsonparser.Array, Val: sb.Bytes()}
}
