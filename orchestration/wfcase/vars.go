package wfcase

import (
	"fmt"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/PaesslerAG/gval"
	"github.com/rs/zerolog/log"
	"regexp"
)

/*
 * ProcessVariables
 */

const (
	SymphonyOrchestrationIdProcessVar          = "smp_orchestration_id"
	SymphonyOrchestrationDescriptionProcessVar = "smp_orchestration_descr"
)

var IsIdentifierRegexp = regexp.MustCompile("^[a-zA-Z_0-9]+(\\.[a-zA-Z_0-9]+)*$")

type ProcessVar struct {
	Name  string      `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Value interface{} `yaml:"value,omitempty" mapstructure:"value,omitempty" json:"value,omitempty"`
}

type ProcessVars map[string]interface{}

func (vs ProcessVars) Set(n string, expr string, resolver *ProcessVarResolver) error {

	val, _, err := varResolver.ResolveVariables(expr, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return err
	}

	val, isExpr := isExpression(val)

	// Was isExpression(val) but in doing this I use the evaluated value and I depend on the value of the variables  with potentially weird values.
	if isExpr {
		gi, err := gval.Evaluate(val, vs)
		if err != nil {
			return err
		}
		vs[n] = gi
	} else {
		vs[n] = val
	}

	return nil
}

/*
	func (vs ProcessVars) Get(n string) (interface{}, bool) {
		v, ok := vs[n]
		return v, ok
	}
*/

type EvaluationMode string

const (
	ExactlyOne = "exactly-one"
	AtLeastOne = "at-least-one"
)

func (vs ProcessVars) Eval(v string) (interface{}, error) {
	return gval.Evaluate(v, vs)
}

func (vs ProcessVars) Lookup(v string, defaultValue interface{}) (interface{}, bool) {
	if v == "" {
		return defaultValue, false
	}

	res, ok := vs[v]
	if !ok {
		return defaultValue, false
	}

	return res, true
}

func (vs ProcessVars) EvalToBool(v string) (bool, error) {

	// The empty expression evaluates to true.
	boolVal := true

	if v != "" {
		exprValue, err := gval.Evaluate(v, vs)
		if err != nil {
			return false, err
		}

		ok := false
		if boolVal, ok = exprValue.(bool); !ok {
			return false, fmt.Errorf("expression %s is not a boolean expression", v)
		}
	}

	return boolVal, nil
}

func (vs ProcessVars) EvalToString(v string) (string, error) {
	const semLogContext = "process-vars::eval-2-string"
	s := ""
	if v != "" {
		exprValue, err := gval.Evaluate(v, vs)
		if err != nil {
			return s, err
		}

		ok := false
		if s, ok = exprValue.(string); ok {
		} else {
			s = fmt.Sprint(exprValue)
			log.Warn().Str("s", s).Str("expr-type", fmt.Sprintf("%T", exprValue)).Msg(semLogContext + " not a string, casted with fmt.Sprint")
		}
		return s, nil
	}

	return "", nil
}

func (vs ProcessVars) IndexOfTrueExpression(varExpressions []string) (int, error) {
	return vs.evalExpressionSetToBool(varExpressions, ExactlyOne)
}

func (vs ProcessVars) IndexOfFirstTrueExpression(varExpressions []string) (int, error) {
	return vs.evalExpressionSetToBool(varExpressions, AtLeastOne)
}

func (vs ProcessVars) evalExpressionSetToBool(varExpressions []string, mode EvaluationMode) (int, error) {

	foundNdx := -1
	for ndx, v := range varExpressions {

		// The empty expression evaluates to true.
		boolVal, err := vs.EvalToBool(v)
		if err != nil {
			return ndx, err
		}

		if boolVal {
			boolVal, err = onTrueEvaluateModeConstraint(foundNdx >= 0, v == "", mode)
			if err != nil {
				return ndx, fmt.Errorf("expression (%s) at  %d and expression (%s) at %d both evaluate and violate the %s mode",
					varExpressions[foundNdx], foundNdx,
					varExpressions[ndx], ndx,
					mode)
			} else {
				// Override the index if is the first or is a non-empty expression. Useful for at-least with default (always true) constraint
				if boolVal {
					foundNdx = ndx
				}
			}
		}
	}

	if foundNdx < 0 {
		return -1, fmt.Errorf("no expression evaluates to true")
	}

	return foundNdx, nil
}

func onTrueEvaluateModeConstraint(isFound bool, isEmpty bool, mode EvaluationMode) (bool, error) {
	if isFound && mode == ExactlyOne {
		return false, fmt.Errorf("expression violate the %s mode", mode)
	}

	if isFound && isEmpty {
		return false, nil
	}

	return true, nil
}
