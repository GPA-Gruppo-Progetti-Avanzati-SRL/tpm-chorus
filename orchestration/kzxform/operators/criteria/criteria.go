package criteria

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/PaesslerAG/gval"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	OperatorEQ = "eq"
	OperatorIn = "in"
)

type CriterionVar struct {
	With operators.JsonReference
	As   string
}

type Criterion interface {
	IsAccepted(value []byte, vars map[string]interface{}) (bool, error)
	IsZero() bool
	GetType() string
}

type noCriterion struct{}

func (c noCriterion) GetType() string {
	return SpecParamCriterionTypeEmpty
}

func (c noCriterion) IsZero() bool {
	return true
}

func (c noCriterion) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	return false, nil
}

type termOperatorCriterionImpl struct {
	AttributeName operators.JsonReference
	Operator      string
	Term          string
}

func (c termOperatorCriterionImpl) GetType() string {
	return SpecParamCriterionTypeTerm
}

func (c termOperatorCriterionImpl) IsZero() bool {
	const semLogContext = "criteria::is-zero"
	return c.AttributeName.IsZero() && c.Term == ""
}

func (c termOperatorCriterionImpl) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	rc := false

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
	return rc, nil
}

type evalCriterionImpl struct {
	Vars       []CriterionVar
	Expression string
}

func (c evalCriterionImpl) GetType() string {
	return SpecParamCriterionTypeExpression
}

func (c evalCriterionImpl) IsZero() bool {
	const semLogContext = "criteria::is-zero"
	return c.Expression == ""
}

func (c evalCriterionImpl) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	rc := false

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
	SpecParamCriterionExpression   = "eval"

	SpecParamCriterionTypeEmpty      = "empty"
	SpecParamCriterionTypeTerm       = "operator-term"
	SpecParamCriterionTypeExpression = "expression-evaluation"
)

func CriterionFromSpec(c interface{}) (Criterion, error) {
	var err error

	if c == nil {
		return noCriterion{}, nil
	}

	attributeName, err := operators.GetJsonReferenceParamFromMap(c, SpecParamCriterionAttributeReference, false)
	if err != nil {
		return noCriterion{}, err
	}

	if !attributeName.IsZero() {
		var criterion termOperatorCriterionImpl
		criterion.AttributeName = attributeName
		criterion.Operator, err = operators.GetStringParamFromMap(c, SpecParamCriterionOperator, false)
		if err != nil {
			return criterion, err
		}

		if criterion.Operator == "" {
			criterion.Operator = OperatorEQ
		}

		criterion.Term, err = operators.GetStringParamFromMap(c, SpecParamCriterionTerm, true)
		if err != nil {
			return criterion, err
		}

		return criterion, nil
	} else {
		var criterion evalCriterionImpl
		vars, err := operators.GetArrayParamFromMap(c, SpecParamCriterionVars, true)
		if err != nil {
			return criterion, err
		}

		for _, v := range vars {
			vr := CriterionVar{}
			vr.With, err = operators.GetJsonReferenceParamFromMap(v, SpecParamCriterionVariableWith, true)
			if err != nil {
				return criterion, err
			}

			vr.As, err = operators.GetStringParamFromMap(v, SpecParamCriterionVariableAs, true)
			if err != nil {
				return criterion, err
			}

			criterion.Vars = append(criterion.Vars, vr)
		}

		criterion.Expression, err = operators.GetStringParamFromMap(c, SpecParamCriterionExpression, true)
		if err != nil {
			return criterion, err
		}

		return criterion, nil
	}
}

func CriteriaFromSpec(spec *transform.Config, specParamName string, required bool) ([]Criterion, error) {
	const semLogContext = "criteria::get-criteria-from-spec"
	filters, err := operators.GetArrayParam(spec, specParamName, required)
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
