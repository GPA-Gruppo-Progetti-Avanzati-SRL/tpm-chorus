package setproperties

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
)

const (
	SpecParamPropertyNameRef    = "name-ref"
	SpecParamPropertyValue      = "value"
	SpecParamPropertyPath       = "path"
	SpecParamPropertyExpression = "expression"
	SpecParamIfMissing          = "if-missing"
	SpecParamProperties         = "properties"
	SpecParamCriterion          = "criterion"
)

type OperatorParams struct {
	Name       operators.JsonReference
	Value      []byte
	Path       operators.JsonReference
	Expression operators.Expression
	IfMissing  bool
	criterion  operators.Criterion
}

func getParamsFromSpec(c interface{}) (OperatorParams, error) {
	var err error
	pcfg := OperatorParams{}

	pcfg.Name, err = operators.GetJsonReferenceParamFromMap(c, SpecParamPropertyNameRef, true)
	if err != nil {
		return pcfg, err
	}

	pv, err := operators.GetParamFromMap(c, SpecParamPropertyValue, false)
	if err != nil {
		return pcfg, err
	}
	if pv != nil {
		pcfg.Value, err = json.Marshal(pv)
		if err != nil {
			return pcfg, err
		}
	}

	pcfg.Path, err = operators.GetJsonReferenceParamFromMap(c, SpecParamPropertyPath, false)
	if err != nil {
		return pcfg, err
	}

	expressionProperty, err := operators.GetParamFromMap(c, SpecParamPropertyExpression, false)
	if err != nil {
		return pcfg, err
	}
	pcfg.Expression, err = operators.NewExpression(expressionProperty)
	if err != nil {
		return pcfg, err
	}

	if pcfg.Path.IsZero() && pcfg.Value == nil && pcfg.Expression.IsZero() {
		err = errors.New("path or value or expression is required")
		return pcfg, err
	}

	pcfg.IfMissing, err = operators.GetBoolParamFromMap(c, SpecParamIfMissing, false)
	if err != nil {
		return pcfg, err
	}

	criterion, err := operators.GetParamFromMap(c, SpecParamCriterion, false)
	if err != nil {
		return pcfg, err
	}

	pcfg.criterion, err = operators.CriterionFromSpec(criterion)
	if err != nil {
		return pcfg, err
	}

	return pcfg, nil
}
