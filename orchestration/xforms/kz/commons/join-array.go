package commons

import (
	"fmt"
	"strings"

	"github.com/qntfy/jsonparser"
	"github.com/rs/zerolog/log"
)

func JoinArray(v any, delim string) interface{} {
	const semLogContext = "operators-funcs::join-array"
	rc := ""

	switch param := v.(type) {
	case ExpressionVariable:
		switch param.Dt {
		case jsonparser.Array:
			rc, _ = concatJsonArrayOfStrings(param.Val, delim)
		case jsonparser.Null:
			fallthrough
		case jsonparser.NotExist:
			log.Info().Str("var-dt", fmt.Sprint(param.Dt)).Msg(semLogContext + " - concat is empty on nulls")
		default:
			log.Warn().Str("var-dt", fmt.Sprint(param.Dt)).Msg(semLogContext + " - function not applicable")
		}

	case []byte:
		log.Warn().Msg(semLogContext + " - []byte not supported yet")
	default:
		log.Trace().Str("data-type", fmt.Sprintf("%T", v)).Msg(semLogContext)
	}

	return rc
}

func concatJsonArrayOfStrings(sourceArray []byte, delim string) (string, error) {
	const semLogContext = "operators-funcs::json-array-length"

	var concat strings.Builder
	var cnt int
	_, err := jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if errParam != nil {
			log.Error().Err(errParam).Msg(semLogContext)
		}
		if cnt > 0 {
			concat.WriteString(delim)
		}
		concat.WriteString(string(value))
		cnt++
	})

	return concat.String(), err
}
