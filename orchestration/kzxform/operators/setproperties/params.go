package setproperties

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
)

const (
	SpecParamPropertyNameRef = "name-ref"
	SpecParamPropertyValue   = "value"
	SpecParamPropertyPath    = "path"
	SpecParamIfMissing       = "if-missing"
	SpecParamProperties      = "properties"
	SpecParamCriterion       = "criterion"
)

type OperatorParams struct {
	Name      operators.JsonReference
	Value     []byte
	Path      operators.JsonReference
	IfMissing bool
	criterion operators.Criterion
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

	pcfg.Value, err = json.Marshal(pv)
	if err != nil {
		return pcfg, err
	}

	pcfg.Path, err = operators.GetJsonReferenceParamFromMap(c, SpecParamPropertyPath, false)
	if err != nil {
		return pcfg, err
	}

	if pcfg.Path.IsZero() && pcfg.Value == nil {
		err = errors.New("path or value is required")
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
