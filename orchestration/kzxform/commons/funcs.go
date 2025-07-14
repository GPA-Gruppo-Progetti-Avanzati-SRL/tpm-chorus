package commons

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs/amt"
)

func GetFuncMap(current map[string]interface{}) map[string]interface{} {
	if current == nil {
		current = make(map[string]interface{})
	}

	current["_lenArray"] = LenArray
	current["_mergeArrays"] = MergeArrays
	current["_sortArray"] = SortArray
	current["_joinArray"] = JoinArray
	current["_now"] = purefuncs.Now
	current["_age"] = purefuncs.Age
	current["_isDate"] = purefuncs.IsDate
	current["_parseDate"] = purefuncs.ParseDate
	current["_parseAndFormatDate"] = purefuncs.ParseAndFmtDate
	current["_dateDiff"] = purefuncs.DateDiff
	current["_printf"] = purefuncs.Printf
	current["_amtNeg"] = amt.AmtNeg
	current["_amtConv"] = amt.AmtConv
	current["_amtCmp"] = amt.AmtCmp
	current["_amtAdd"] = amt.AmtAdd
	current["_amtDiff"] = amt.AmtDiff
	current["_amtFmt"] = amt.Format
	current["_padLeft"] = purefuncs.PadLeft
	current["_left"] = purefuncs.Left
	current["_right"] = purefuncs.Right
	current["_len"] = purefuncs.Len
	current["_substr"] = purefuncs.Substr
	current["_isDef"] = purefuncs.IsDefined
	current["_b64"] = purefuncs.Base64
	current["_uuid"] = purefuncs.Uuid
	current["_regexMatch"] = purefuncs.RegexMatch
	current["_regexExtractFirst"] = purefuncs.RegexExtractFirst
	current["_regexMatchSetMatchAndExtract"] = purefuncs.RegexSetMatchAndExtract
	current["_lenJsonArray"] = purefuncs.LenJsonArray
	current["_isJsonArray"] = purefuncs.IsJsonArray
	current["_stringIn"] = purefuncs.StringIn
	current["_trimSpace"] = purefuncs.TrimSpace
	current["_hashPartition"] = purefuncs.HashPartition
	return current
}
