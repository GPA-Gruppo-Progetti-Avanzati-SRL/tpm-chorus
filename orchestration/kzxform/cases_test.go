package kzxform_test

import _ "embed"

type TestInfo struct {
	ruleId   string
	input    []byte
	outputFn string
}

var tests = map[string]TestInfo{
	addArrayItems000.ruleId:    addArrayItems000,
	mergeArraysItems000.ruleId: mergeArraysItems000,
	mergeArraysItems001.ruleId: mergeArraysItems001,
}

//go:embed case-001-input.json
var case001Input []byte

//go:embed case-001-rule.yml
var case001RuleYml []byte

//go:embed case-002-input.json
var case002Input []byte

//go:embed case-002-rule.yml
var case002RuleYml []byte

//go:embed case-003-input.json
var case003Input []byte

//go:embed case-003b-input.json
var case003bInput []byte

//go:embed case-003-rule.yml
var case003RuleYml []byte

//go:embed case-003b-rule.yml
var case003bRuleYml []byte

//go:embed case-004-input.json
var case004Input []byte

//go:embed case-004-rule.yml
var case004RuleYml []byte

//go:embed case-005-input.json
var case005Input []byte

//go:embed case-005-rule.yml
var case005RuleYml []byte

//go:embed case-006-input.json
var case006Input []byte

//go:embed case-006-rule.yml
var case006RuleYml []byte

//go:embed case-007-input.json
var case007Input []byte

//go:embed case-007-rule.yml
var case007RuleYml []byte

//go:embed case-008-input.json
var case008Input []byte

//go:embed case-008-rule.yml
var case008RuleYml []byte

//go:embed case-009-input.json
var case009Input []byte

//go:embed case-009-rule.yml
var case009RuleYml []byte

//go:embed case-010-input.json
var case010Input []byte

//go:embed case-010-rule.yml
var case010RuleYml []byte

//go:embed case-011-input.json
var case011Input []byte

//go:embed case-011-rule.yml
var case011RuleYml []byte

//go:embed case-012-input.json
var case012Input []byte

//go:embed case-012-rule.yml
var case012RuleYml []byte

//go:embed case-013-input.json
var case013Input []byte

//go:embed case-013-rule.yml
var case013RuleYml []byte

//go:embed set-properties-000-input.json
var set_properties_000_input []byte

//go:embed set-properties-000.yml
var set_properties_000 []byte

//go:embed filter-array-items-000-input.json
var filter_array_items_000_input []byte

//go:embed  filter-array-items-000.yml
var filter_array_items_000 []byte

//go:embed merge-arrays-000-input.json
var merge_arrays_000_input []byte

//go:embed  merge-arrays-000.yml
var merge_arrays_items_000 []byte

var mergeArraysItems000 = TestInfo{
	"merge_arrays_000",
	merge_arrays_000_input,
	"amerge-arrays-000-output.json",
}

//go:embed merge-arrays-001-input.json
var merge_arrays_001_input []byte

//go:embed  merge-arrays-001.yml
var merge_arrays_items_001 []byte

var mergeArraysItems001 = TestInfo{
	"merge_arrays_001",
	merge_arrays_001_input,
	"merge-arrays-001-output.json",
}

//go:embed add-array-items-000-input.json
var add_arrays_000_input []byte

//go:embed  add-array-items-000.yml
var add_arrays_items_000 []byte

var addArrayItems000 = TestInfo{
	"add_array_items_000",
	add_arrays_000_input,
	"add-array-items-000-output.json",
}
