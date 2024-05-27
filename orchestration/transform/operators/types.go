package operators

const (
	SpecParamSourceReference             = "source-ref"
	SpecParamTargetReference             = "target-ref"
	SpecParamCriteria                    = "criteria"
	SpecParamCriterionAttributeReference = "attribute-ref"
	SpecParamCriterionOperator           = "operator"
	SpecParamCriterionTerm               = "term"
	SpecParamPropertyNameRef             = "name-ref"
	SpecParamPropertyValueRef            = "value-ref"
	SpecParamProperties                  = "properties"
	SpecParamPropertyValue               = "value"
	SpecParamSubRules                    = "sub-rules"
	SpecParamConversions                 = "conversions"
	SpecParamConversionType              = "type"
	SpecParamSourceUnit                  = "source-unit"
	SpecParamTargetUnit                  = "target-unit"
	SpecParamDecimalFormat               = "decimal-format"
	SpecParamIfMissing                   = "if-missing"
	OperatorsTempReusltPropertyName      = "smp-tmp"
)

type JsonReference struct {
	Path string
	Keys []string
}

func ToJsonReference(s string) (JsonReference, error) {
	keys, err := SplitKeySpecifier(s)
	if err != nil {
		return JsonReference{}, err
	}

	jr := JsonReference{
		Path: s,
		Keys: keys,
	}

	return jr, nil
}

func (jr JsonReference) IsZero() bool {
	return jr.Path == ""
}
