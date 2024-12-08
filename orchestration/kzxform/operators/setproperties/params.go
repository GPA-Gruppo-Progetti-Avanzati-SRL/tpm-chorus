package setproperties

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
)

const (
	SpecParamPropertyNameRef = "name-ref"
	SpecParamPropertyValue   = "value"
	SpecParamIfMissing       = "if-missing"
	SpecParamProperties      = "properties"
)

type OperatorParams struct {
	Name      operators.JsonReference
	Value     []byte
	IfMissing bool
}

func getParamsFromSpec(c interface{}) (OperatorParams, error) {
	var err error
	pcfg := OperatorParams{}

	pcfg.Name, err = operators.GetJsonReferenceParamFromMap(c, SpecParamPropertyNameRef, true)
	if err != nil {
		return pcfg, err
	}

	pv, err := operators.GetParamFromMap(c, SpecParamPropertyValue, true)
	if err != nil {
		return pcfg, err
	}

	pcfg.Value, err = json.Marshal(pv)
	if err != nil {
		return pcfg, err
	}

	pcfg.IfMissing, err = operators.GetBoolParamFromMap(c, SpecParamIfMissing, false)
	if err != nil {
		return pcfg, err
	}

	return pcfg, nil
}
