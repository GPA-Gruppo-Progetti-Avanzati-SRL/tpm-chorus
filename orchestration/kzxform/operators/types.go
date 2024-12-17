package operators

import (
	"fmt"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

type JsonReference struct {
	WithArrayISpecifierIndex    int
	WithArrayPushSpecifierIndex int
	Path                        string
	Keys                        []string
}

func MustToJsonReference(s string) JsonReference {
	j, err := ToJsonReference(s)
	if err != nil {
		panic(err)
	}

	return j
}

func ToJsonReference(s string) (JsonReference, error) {
	keys, iSpecifierNdx, plusNdx, err := SplitKeySpecifier(s)
	if err != nil {
		return JsonReference{}, err
	}

	jr := JsonReference{
		Path:                        s,
		Keys:                        keys,
		WithArrayISpecifierIndex:    iSpecifierNdx,
		WithArrayPushSpecifierIndex: plusNdx,
	}

	return jr, nil
}

func (jr JsonReference) IsZero() bool {
	return jr.Path == ""
}

func (jr JsonReference) JsonReferenceToArrayWithIotaSpecifier() JsonReference {
	rootRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     jr.Path[:strings.Index(jr.Path, "[i]")],
		Keys:                     jr.Keys[:jr.WithArrayISpecifierIndex],
	}

	return rootRef
}

func (jr JsonReference) JsonReferenceToArrayItemWithIotaSpecifier(i int) JsonReference {
	rootRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     strings.ReplaceAll(jr.Path, "[i]", fmt.Sprintf("[%d]", i)),
	}
	rootRef.Keys = append(rootRef.Keys, jr.Keys...)
	rootRef.Keys[jr.WithArrayISpecifierIndex] = fmt.Sprintf("[%d]", i)
	return rootRef
}

func (jr JsonReference) JsonReferenceToArrayNestedItemWithIotaSpecifierBoh(i int) JsonReference {
	nestedRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     strings.ReplaceAll(jr.Path[strings.Index(jr.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", i)),
	}
	nestedRef.Keys = append(nestedRef.Keys, jr.Keys[jr.WithArrayISpecifierIndex:]...)
	nestedRef.Keys[0] = fmt.Sprintf("[%d]", i)
	return nestedRef
}

func (jr JsonReference) JsonReferenceToArrayNestedItemWithIotaSpecifier(i int) JsonReference {
	nestedRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     jr.Path[strings.Index(jr.Path, "[i]")+len("[i]")+1:],
	}
	nestedRef.Keys = append(nestedRef.Keys, jr.Keys[jr.WithArrayISpecifierIndex+1:]...)
	return nestedRef
}

const (
	OperatorEQ = "eq"
	OperatorIn = "in"
)

type Criterion struct {
	AttributeName JsonReference
	Operator      string
	Term          string
}

func (c Criterion) IsZero() bool {
	return c.AttributeName.IsZero() && c.Term == ""
}

func (c Criterion) IsAccepted(value []byte) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	attributeValue, dataType, _, err := jsonparser.Get(value, c.AttributeName.Keys...)
	if err != nil {
		return false, err
	}

	if dataType == jsonparser.NotExist {
		return false, nil
	}

	term := strings.ToLower(c.Term)
	attrValue := strings.ToLower(string(attributeValue))
	rc := false
	switch c.Operator {
	case OperatorIn:
		if strings.Contains(term, attrValue) {
			rc = true
		}
	case OperatorEQ:
		if attrValue == term {
			rc = true
		}
	default:
		if attrValue == term {
			rc = true
		}
	}

	return rc, nil
}

type Criteria []Criterion

func (ca Criteria) IsAccepted(value []byte) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	for _, criterion := range ca {

		ok, err := criterion.IsAccepted(value)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

const (
	SpecParamCriterionAttributeReference = "attribute-ref"
	SpecParamCriterionTerm               = "term"
	SpecParamCriterionOperator           = "operator"
)

func CriterionFromSpec(c interface{}) (Criterion, error) {
	var err error
	var criterion Criterion
	criterion.AttributeName, err = GetJsonReferenceParamFromMap(c, SpecParamCriterionAttributeReference, true)
	if err != nil {
		return criterion, err
	}

	criterion.Operator, err = GetStringParamFromMap(c, SpecParamCriterionOperator, true)
	if err != nil {
		return criterion, err
	}

	criterion.Term, err = GetStringParamFromMap(c, SpecParamCriterionTerm, true)
	if err != nil {
		return criterion, err
	}

	return criterion, nil
}

func CriteriaFromSpec(spec *transform.Config, specParamName string) ([]Criterion, error) {
	const semLogContext = "criteria::get-criteria-from-spec"
	filters, err := GetArrayParam(spec, specParamName, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	if len(filters) == 0 {
		return nil, nil
	}

	filtersObj := make([]Criterion, 0)
	for _, f := range filters {
		crit, err := CriterionFromSpec(f)
		if err != nil {
			return nil, err
		}

		filtersObj = append(filtersObj, crit)
	}

	return filtersObj, nil
}
