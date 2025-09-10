package shiftarrayitems

import (
	"encoding/json"

	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	SpecParamSourceReference             = "source-ref"
	SpecParamTargetReference             = "target-ref"
	SpecParamCriteria                    = "criteria"
	SpecParamCriterionAttributeReference = "attribute-ref"
	SpecParamCriterionTerm               = "term"
	SpecParamSubRules                    = "sub-rules"
	OperatorsTempResultPropertyName      = "smp-tmp"
	SpecParamFilterItems                 = "filter-items"
	SpecParamFlatten                     = "flatten"
)

/*
	type criterion struct {
		attributeName operators.JsonReference
		operator      string
		term          string
	}

	func getCriterionFromSpec(c interface{}) (operators.Criterion, error) {
		var err error
		var criterion operators.Criterion
		criterion.AttributeName, err = operators.GetJsonReferenceParamFromMap(c, SpecParamCriterionAttributeReference, true)
		if err != nil {
			return criterion, err
		}

		criterion.Operator = "eq"

		criterion.Term, err = operators.GetStringParamFromMap(c, SpecParamCriterionTerm, true)
		if err != nil {
			return criterion, err
		}

		return criterion, nil
	}

	func getCriteriaFromSpec(c []interface{}) ([]operators.Criterion, error) {
		filtersObj := make([]operators.Criterion, 0)
		for _, f := range c {
			crit, err := getCriterionFromSpec(f)
			if err != nil {
				return nil, err
			}

			filtersObj = append(filtersObj, crit)
		}

		return filtersObj, nil
	}
*/

type OperatorParams struct {
	sourceRef           operators2.JsonReference
	destRef             operators2.JsonReference
	filterItems         bool
	flatten             bool
	inPlace             bool
	criteria            operators2.Criteria
	itemRulesSerialized string
}

func getParamsFromSpec(spec *transform.Config) (OperatorParams, error) {
	const semLogContext = OperatorSemLogContext + "::get-params-from-specs"
	var err error

	params := OperatorParams{}

	params.sourceRef, err = operators2.GetJsonReferenceParam(spec, SpecParamSourceReference, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.destRef, err = operators2.GetJsonReferenceParam(spec, SpecParamTargetReference, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.inPlace = false
	if params.destRef.IsZero() || params.sourceRef.Path == params.destRef.Path {
		params.destRef = params.sourceRef
		params.inPlace = true
	}

	params.filterItems, err = operators2.GetBoolParam(spec, SpecParamFilterItems, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.flatten, err = operators2.GetBoolParam(spec, SpecParamFlatten, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	itemRules, err := operators2.GetArrayParam(spec, SpecParamSubRules, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	log.Debug().Str(SpecParamTargetReference, params.sourceRef.Path).Bool("in-place", params.inPlace).Msg(semLogContext)

	if itemRules != nil {
		b, err := json.Marshal(itemRules)
		if err != nil {
			return params, err
		}

		params.itemRulesSerialized = string(b)
		log.Debug().Str(SpecParamSubRules, params.itemRulesSerialized).Msg(semLogContext)
	}

	params.criteria, err = operators2.CriteriaFromSpec(spec, SpecParamCriteria, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	/*
		    This implementation supported only the old style..
			filters, err := operators.GetArrayParam(spec, SpecParamCriteria, false)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return params, err
			}

			if filters != nil {
				params.criteria, err = getCriteriaFromSpec(filters)
			}
	*/

	log.Debug().
		Interface(SpecParamSourceReference, params.sourceRef).
		Interface(SpecParamTargetReference, params.destRef).
		Interface(SpecParamCriteria, params.criteria).
		Bool("in-place", params.inPlace).
		Msg(semLogContext)

	return params, nil
}
