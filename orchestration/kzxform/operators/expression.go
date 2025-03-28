package operators

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/commons"
	"github.com/PaesslerAG/gval"
	"github.com/qntfy/jsonparser"
	"github.com/rs/zerolog/log"
)

const (
	ExpressionSpecParamVariableWith           = "with"
	ExpressionSpecParamVariableAs             = "as"
	ExpressionSpecParamVariableIfMissingValue = "defaults-to"
	ExpressionSpecParamVars                   = "vars"
	ExpressionSpecParamExpressionText         = "text"
)

type ExpressionVar struct {
	With           JsonReference
	As             string
	IfMissingValue interface{}
}

type Expression interface {
	String() string
	IsZero() bool
	Eval(value []byte, vars map[string]interface{}) (interface{}, error)
}

type noExpression struct{}

func (c noExpression) IsZero() bool {
	return true
}

func (c noExpression) Eval(value []byte, vars map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (c noExpression) String() string {
	return "[no-expression]"
}

type expressionImpl struct {
	Vars []ExpressionVar
	Text string
}

func NewExpression(c interface{}) (Expression, error) {
	var exp expressionImpl
	vars, err := GetArrayParamFromMap(c, ExpressionSpecParamVars, true)
	if err != nil {
		return exp, err
	}

	for _, v := range vars {
		vr := ExpressionVar{}
		vr.With, err = GetJsonReferenceParamFromMap(v, ExpressionSpecParamVariableWith, true)
		if err != nil {
			return exp, err
		}

		vr.As, err = GetStringParamFromMap(v, ExpressionSpecParamVariableAs, true)
		if err != nil {
			return exp, err
		}

		vr.IfMissingValue, err = GetParamFromMap(v, ExpressionSpecParamVariableIfMissingValue, false)
		if err != nil {
			return exp, err
		}

		exp.Vars = append(exp.Vars, vr)
	}

	exp.Text, err = GetStringParamFromMap(c, ExpressionSpecParamExpressionText, true)
	if err != nil {
		return exp, err
	}

	return exp, nil
}

func (c expressionImpl) IsZero() bool {
	const semLogContext = "expression::is-zero"
	return c.Text == ""
}

func (c expressionImpl) String() string {
	return c.Text
}

func (c expressionImpl) Eval(value []byte, vars map[string]interface{}) (interface{}, error) {
	const semLogContext = "expression::eval"

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

	result, err := gval.Evaluate(c.Text, mapOfVars)
	if err != nil {
		return nil, err
	}

	return result, nil
}
