package wfcase

import (
	"fmt"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/PaesslerAG/gval"
)

/*
 * ProcessVariables
 */

const (
	SymphonyOrchestrationIdProcessVar          = "smp_orchestration_id"
	SymphonyOrchestrationDescriptionProcessVar = "smp_orchestration_descr"
)

type ProcessVar struct {
	Name  string      `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Value interface{} `yaml:"value,omitempty" mapstructure:"value,omitempty" json:"value,omitempty"`
}

type ProcessVars map[string]interface{}

func (vs ProcessVars) Set(n string, expr string, resolver *ProcessVarResolver) error {

	isExpr := isExpression(expr)

	val, _, err := varResolver.ResolveVariables(expr, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return err
	}

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

func (vs ProcessVars) Get(n string) (interface{}, bool) {
	v, ok := vs[n]
	return v, ok
}

type EvaluationMode string

const (
	ExactlyOne = "exactly-one"
	AtLeastOne = "at-least-one"
)

func (vs ProcessVars) Eval(varExpressions []string, mode EvaluationMode) (int, error) {

	foundNdx := -1
	for ndx, v := range varExpressions {

		// The empty expression evaluates to true.
		boolVal, err := vs.BoolEval(v)
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

func (vs ProcessVars) BoolEval(v string) (bool, error) {

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
