package setproperties

import (
	"encoding/json"
	"errors"

	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/rs/zerolog/log"
)

const (
	SpecParamPropertyNameRef    = "name-ref"
	SpecParamPropertyValue      = "value"
	SpecParamPropertyPath       = "path"
	SpecParamPropertyExpression = "expression"
	SpecParamIfMissing          = "if-missing"
	SpecParamProperties         = "properties"
	SpecParamCriterion          = "criterion"
	SpecParamPropertyFormat     = "format"
)

type OperatorParams struct {
	Name       operators2.JsonReference
	Value      []byte
	Path       operators2.JsonReference
	Format     string
	Expression operators2.Expression
	IfMissing  bool
	criterion  operators2.Criterion
}

func getParamsFromSpec(c interface{}) (OperatorParams, error) {
	const semLogContext = OperatorSemLogContext + "::get-params-from-spec"
	var err error
	pcfg := OperatorParams{}

	pcfg.Name, err = operators2.GetJsonReferenceParamFromMap(c, SpecParamPropertyNameRef, true)
	if err != nil {
		return pcfg, err
	}

	pv, err := operators2.GetParamFromMap(c, SpecParamPropertyValue, false)
	if err != nil {
		return pcfg, err
	}
	if pv != nil {
		pcfg.Value, err = json.Marshal(pv)
		if err != nil {
			return pcfg, err
		}
	}

	pcfg.Path, err = operators2.GetJsonReferenceParamFromMap(c, SpecParamPropertyPath, false)
	if err != nil {
		return pcfg, err
	}

	pcfg.Format, err = operators2.GetStringParamFromMap(c, SpecParamPropertyFormat, false)
	if err != nil {
		return pcfg, err
	}

	expressionProperty, err := operators2.GetParamFromMap(c, SpecParamPropertyExpression, false)
	if err != nil {
		return pcfg, err
	}
	pcfg.Expression, err = operators2.NewExpression(expressionProperty)
	if err != nil {
		return pcfg, err
	}

	if pcfg.Path.IsZero() && pcfg.Value == nil && pcfg.Expression.IsZero() {
		err = errors.New("path or value or expression is required")
		log.Warn().Err(err).Msg(semLogContext)
		// return pcfg, err
	}

	pcfg.IfMissing, _, err = operators2.GetBoolParamFromMap(c, SpecParamIfMissing, false)
	if err != nil {
		return pcfg, err
	}

	criterion, err := operators2.GetParamFromMap(c, SpecParamCriterion, false)
	if err != nil {
		return pcfg, err
	}

	pcfg.criterion, err = operators2.CriterionFromSpec(criterion)
	if err != nil {
		return pcfg, err
	}

	return pcfg, nil
}
