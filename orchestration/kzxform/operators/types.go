package operators

import (
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	CriterionSystemVariableKazaamArrayLen       = "_kz_array_len"
	CriterionSystemVariableKazaamArrayLoopIndex = "_kz_array_ndx"
)

type JsonReference struct {
	WithArrayISpecifierIndex    int
	WithArrayPushSpecifierIndex int
	Path                        string
	IsPathRelative              bool
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
	isAbs := false
	if strings.HasPrefix(s, "/") {
		isAbs = true
	}

	s = strings.TrimPrefix(s, "./")
	s = strings.TrimSuffix(s, "/")

	keys, iSpecifierNdx, plusNdx, err := SplitKeySpecifier(s)
	if err != nil {
		return JsonReference{}, err
	}

	jr := JsonReference{
		Path:                        s,
		IsPathRelative:              !isAbs,
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

type CriterionVar struct {
	With JsonReference
	As   string
}

type Criterion struct {
	Typ           string
	AttributeName JsonReference
	Operator      string
	Term          string

	Vars       []CriterionVar
	Expression string
}

func (c Criterion) IsZero() bool {
	const semLogContext = "criteria::is-zero"

	if c.Typ == "" {
		return true
	}

	if c.Typ == SpecParamCriterionTypeTerm {
		return c.AttributeName.IsZero() && c.Term == ""
	}

	if c.Typ == SpecParamCriterionTypeExpression {
		return c.Expression == ""
	}

	err := errors.New("unknown criterion type")
	log.Error().Err(err).Msg(semLogContext)
	return false
}

func (c Criterion) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	rc := false
	if c.Typ == SpecParamCriterionTypeTerm {
		attributeValue, dataType, _, err := jsonparser.Get(value, c.AttributeName.Keys...)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return false, err
		}

		if dataType == jsonparser.NotExist {
			return false, nil
		}

		term := strings.ToLower(c.Term)
		attrValue := strings.ToLower(string(attributeValue))
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
	} else {
		mapOfVars := vars
		if mapOfVars == nil {
			mapOfVars = make(map[string]interface{})
		}
		for _, v := range c.Vars {
			varValue, dataType, _, err := jsonparser.Get(value, v.With.Keys...)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return false, err
			}

			switch dataType {
			case jsonparser.NotExist:
				mapOfVars[v.As] = ""
			case jsonparser.String:
				mapOfVars[v.As] = string(varValue)
			default:
				mapOfVars[v.As] = fmt.Sprint(varValue)
			}
		}

		result, err := gval.Evaluate(c.Expression, mapOfVars)
		if err != nil {
			return false, err
		}

		if b, ok := result.(bool); ok {
			rc = b
		} else {
			log.Error().Err(err).Interface("result", result).Msg(semLogContext)
			rc = false
		}
	}

	return rc, nil
}

type Criteria []Criterion

func (ca Criteria) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	for _, criterion := range ca {

		ok, err := criterion.IsAccepted(value, vars)
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

	SpecParamCriterionTerm     = "term"
	SpecParamCriterionOperator = "operator"

	SpecParamCriterionVariableWith = "with"
	SpecParamCriterionVariableAs   = "as"
	SpecParamCriterionVars         = "vars"
	SpecParamCriterionExpression   = "expression"

	SpecParamCriterionTypeTerm       = "operator-term"
	SpecParamCriterionTypeExpression = "expression-evaluation"
)

func CriterionFromSpec(c interface{}) (Criterion, error) {
	var err error
	var criterion Criterion
	criterion.AttributeName, err = GetJsonReferenceParamFromMap(c, SpecParamCriterionAttributeReference, false)
	if err != nil {
		return criterion, err
	}

	if !criterion.AttributeName.IsZero() {
		criterion.Typ = SpecParamCriterionTypeTerm
		criterion.Operator, err = GetStringParamFromMap(c, SpecParamCriterionOperator, false)
		if err != nil {
			return criterion, err
		}

		if criterion.Operator == "" {
			criterion.Operator = OperatorEQ
		}

		criterion.Term, err = GetStringParamFromMap(c, SpecParamCriterionTerm, true)
		if err != nil {
			return criterion, err
		}
	} else {
		criterion.Typ = SpecParamCriterionTypeExpression
		vars, err := GetArrayParamFromMap(c, SpecParamCriterionVars, true)
		if err != nil {
			return criterion, err
		}

		for _, v := range vars {
			vr := CriterionVar{}
			vr.With, err = GetJsonReferenceParamFromMap(v, SpecParamCriterionVariableWith, true)
			if err != nil {
				return criterion, err
			}

			vr.As, err = GetStringParamFromMap(v, SpecParamCriterionVariableAs, true)
			if err != nil {
				return criterion, err
			}

			criterion.Vars = append(criterion.Vars, vr)
		}

		criterion.Expression, err = GetStringParamFromMap(c, SpecParamCriterionExpression, true)
		if err != nil {
			return criterion, err
		}
	}

	return criterion, nil
}

func CriteriaFromSpec(spec *transform.Config, specParamName string, required bool) ([]Criterion, error) {
	const semLogContext = "criteria::get-criteria-from-spec"
	filters, err := GetArrayParam(spec, specParamName, required)
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
