package commons

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs"
)

func GetFuncMap(current map[string]interface{}) map[string]interface{} {
	if current == nil {
		current = make(map[string]interface{})
	}

	current["_lenArray"] = LenArray
	current["_mergeArrays"] = MergeArrays
	current["_sortArray"] = SortArray
	current["_regexMatchSetMatchAndExtract"] = purefuncs.RegexSetMatchAndExtract
	return current
}
