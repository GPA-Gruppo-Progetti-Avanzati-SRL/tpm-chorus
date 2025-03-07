package operators

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/commons"
	"github.com/PaesslerAG/gval"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	CriterionOperatorEQ = "eq"
	CriterionOperatorIn = "in"

	CriterionSpecParamAttributeReference = "attribute-ref"

	CriterionSpecParamTerm     = "term"
	CriterionSpecParamOperator = "operator"

	CriterionSpecParamVariableWith           = "with"
	CriterionSpecParamVariableAs             = "as"
	CriterionSpecParamVars                   = "vars"
	CriterionSpecParamExpressionText         = "text"
	CriterionSpecParamVariableIfMissingValue = "defaults-to"

	CriterionTypeEmpty      = "empty"
	CriterionTypeTerm       = "operator-term"
	CriterionTypeExpression = "expression"
)

type CriterionVar struct {
	With           JsonReference
	As             string
	IfMissingValue interface{}
}

type Criterion interface {
	IsAccepted(value []byte, vars map[string]interface{}) (bool, error)
	IsZero() bool
	GetType() string
}

type noCriterion struct{}

func (c noCriterion) GetType() string {
	return CriterionTypeEmpty
}

func (c noCriterion) IsZero() bool {
	return true
}

func (c noCriterion) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	return false, nil
}

type termOperatorCriterionImpl struct {
	AttributeName JsonReference
	Operator      string
	Term          string
}

func newTermOperatorCriterionImpl(c interface{}) (termOperatorCriterionImpl, error) {
	var err error
	var criterion termOperatorCriterionImpl
	criterion.AttributeName, err = GetJsonReferenceParamFromMap(c, CriterionSpecParamAttributeReference, true)
	if err != nil {
		return termOperatorCriterionImpl{}, err
	}

	criterion.Operator, err = GetStringParamFromMap(c, CriterionSpecParamOperator, false)
	if err != nil {
		return criterion, err
	}

	if criterion.Operator == "" {
		criterion.Operator = CriterionOperatorEQ
	}

	criterion.Term, err = GetStringParamFromMap(c, CriterionSpecParamTerm, true)
	if err != nil {
		return criterion, err
	}

	return criterion, nil
}

func (c termOperatorCriterionImpl) GetType() string {
	return CriterionTypeTerm
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
	case CriterionOperatorIn:
		if strings.Contains(term, attrValue) {
			rc = true
		}
	case CriterionOperatorEQ:
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

func newEvalCriterionImpl(c interface{}) (evalCriterionImpl, error) {
	var criterion evalCriterionImpl
	vars, err := GetArrayParamFromMap(c, CriterionSpecParamVars, true)
	if err != nil {
		return criterion, err
	}

	for _, v := range vars {
		vr := CriterionVar{}
		vr.With, err = GetJsonReferenceParamFromMap(v, CriterionSpecParamVariableWith, true)
		if err != nil {
			return criterion, err
		}

		vr.As, err = GetStringParamFromMap(v, CriterionSpecParamVariableAs, true)
		if err != nil {
			return criterion, err
		}

		vr.IfMissingValue, err = GetParamFromMap(v, CriterionSpecParamVariableIfMissingValue, false)
		if err != nil {
			return criterion, err
		}

		criterion.Vars = append(criterion.Vars, vr)
	}

	criterion.Expression, err = GetStringParamFromMap(c, CriterionSpecParamExpressionText, true)
	if err != nil {
		return criterion, err
	}

	return criterion, nil
}

func (c evalCriterionImpl) GetType() string {
	return CriterionTypeExpression
}

func (c evalCriterionImpl) IsZero() bool {
	const semLogContext = "criteria::is-zero"
	return c.Expression == ""
}

func (c evalCriterionImpl) IsAccepted(value []byte, vars map[string]interface{}) (bool, error) {
	const semLogContext = "criteria::is-accepted"

	rc := false

	mapOfVars := commons.GetFuncMap(vars)
	for _, v := range c.Vars {
		varValue, dataType, _, err := jsonparser.Get(value, v.With.Keys...)
		if err != nil {
			if dataType == jsonparser.NotExist && v.IfMissingValue != nil {
				switch v.IfMissingValue.(type) {
				case string:
					dataType = jsonparser.String
					varValue = []byte(v.IfMissingValue.(string))
				case float64:
					dataType = jsonparser.Number
					varValue = []byte(fmt.Sprint(v.IfMissingValue.(float64)))
				default:
					log.Error().Err(err).Str("of-type", fmt.Sprintf("%T", value)).Str("value", fmt.Sprintf("%v", value)).Msg(semLogContext)
					return false, err
				}
				log.Warn().Err(err).Str("value", string(value)).Msg(semLogContext)
			} else {
				return false, err
			}
		}

		switch dataType {
		case jsonparser.NotExist:
			mapOfVars[v.As] = ""
		case jsonparser.String:
			mapOfVars[v.As] = string(varValue)
		default:
			expressionVariable := commons.NewExpressionVariable(varValue, dataType)
			mapOfVars[v.As] = expressionVariable
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

func CriterionFromSpec(c interface{}) (Criterion, error) {
	var err error

	if c == nil {
		return noCriterion{}, nil
	}

	attributeName, err := GetJsonReferenceParamFromMap(c, CriterionSpecParamAttributeReference, false)
	if err != nil {
		return noCriterion{}, err
	}

	if !attributeName.IsZero() {
		var criterion termOperatorCriterionImpl
		criterion, err = newTermOperatorCriterionImpl(c)
		if err != nil {
			return criterion, err
		}

		return criterion, nil
	} else {
		var criterion evalCriterionImpl
		criterion, err = newEvalCriterionImpl(c)
		if err != nil {
			return criterion, err
		}

		return criterion, nil
	}
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
