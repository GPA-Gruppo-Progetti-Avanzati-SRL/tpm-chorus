package commons

import (
	"fmt"

	"github.com/qntfy/jsonparser"
	"github.com/rs/zerolog/log"
)

const (
	LenFunctionFormatNumber = "number"
	LenFunctionFormatString = "string"
	//LenFunctionFormatVariableNumber = "v-number"
	//LenFunctionFormatVariableString = "v-string"
)

func LenArray(v any, format string) interface{} {
	const semLogContext = "operators-funcs::array-length"
	rc := 0

	switch param := v.(type) {
	case ExpressionVariable:
		switch param.Dt {
		case jsonparser.Array:
			rc, _ = lenOfJsonArray(param.Val)
		case jsonparser.Null:
			fallthrough
		case jsonparser.NotExist:
			log.Info().Str("var-dt", fmt.Sprint(param.Dt)).Msg(semLogContext + " - len is 0 on nulls")
		default:
			log.Warn().Str("var-dt", fmt.Sprint(param.Dt)).Msg(semLogContext + " - function not applicable")
		}

	case []byte:
		log.Warn().Msg(semLogContext + " - []byte not supported yet")
	default:
		log.Trace().Str("data-type", fmt.Sprintf("%T", v)).Msg(semLogContext)
	}

	switch format {
	case LenFunctionFormatString:
		return fmt.Sprint(rc)
	case LenFunctionFormatNumber:
		return rc
	//case LenFunctionFormatVariableNumber:
	//	return ExpressionVariable{Dt: jsonparser.Number, Val: []byte(fmt.Sprint(1))}
	//case LenFunctionFormatVariableString:
	//	return ExpressionVariable{Dt: jsonparser.String, Val: []byte(fmt.Sprint(1))}
	default:
		return ExpressionVariable{Dt: jsonparser.String, Val: []byte(fmt.Sprint(rc))}
	}
}

func lenOfJsonArray(sourceArray []byte) (int, error) {
	const semLogContext = "operators-funcs::json-array-length"

	var loopNdx int
	_, err := jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if errParam != nil {
			log.Error().Err(errParam).Msg(semLogContext)
		}
		loopNdx++
	})

	return loopNdx, err
}
