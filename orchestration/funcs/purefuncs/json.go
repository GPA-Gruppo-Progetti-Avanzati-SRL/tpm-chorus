package purefuncs

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
)

func LenJsonArray(variable interface{}) int {
	const semLogContext = "orchestration-funcs::len-json-array"

	if util.IsNilish(variable) {
		return 0
	}

	if arr, ok := variable.([]interface{}); ok {
		return len(arr)
	} else {
		log.Warn().Msg(semLogContext + " variable is not array")
	}
	return 0
}

func IsJsonArray(variable interface{}) bool {
	const semLogContext = "orchestration-funcs::is-json-array"

	if util.IsNilish(variable) {
		return false
	}

	if _, ok := variable.([]interface{}); ok {
		return true
	}

	return false
}
