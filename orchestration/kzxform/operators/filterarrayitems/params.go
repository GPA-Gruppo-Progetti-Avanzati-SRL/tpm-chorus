package filterarrayitems

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators/criteria"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	SpecParamSourceReference             = "source-ref"
	SpecParamTargetReference             = "target-ref"
	SpecParamCriteria                    = "criteria"
	SpecParamCriterionAttributeReference = "attribute-ref"
	SpecParamCriterionTerm               = "term"
	OperatorsTempReusltPropertyName      = "smp-tmp"
)

type FilterArrayParams struct {
	sourceRef operators.JsonReference
	destRef   operators.JsonReference
	inPlace   bool
	criteria  criteria.Criteria
}

func getFilterParamsFromSpec(spec *transform.Config) (FilterArrayParams, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"
	var err error

	params := FilterArrayParams{}

	params.sourceRef, err = operators.GetJsonReferenceParam(spec, SpecParamSourceReference, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.destRef, err = operators.GetJsonReferenceParam(spec, SpecParamTargetReference, false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	params.inPlace = false
	if params.destRef.IsZero() || params.sourceRef.Path == params.destRef.Path {
		params.destRef = params.sourceRef
		params.inPlace = true
	}

	params.criteria, err = criteria.CriteriaFromSpec(spec, SpecParamCriteria, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	/*
		filters, err := operators.GetArrayParam(spec, SpecParamCriteria, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return params, err
		}

		params.criteria, err = getFilterConfigsFromSpec(filters)
	*/
	log.Debug().Interface(SpecParamSourceReference, params.sourceRef).Interface(SpecParamTargetReference, params.destRef).Interface(SpecParamCriteria, params.criteria).Bool("in-place", params.inPlace).Msg(semLogContext)
	return params, nil

}

/*
type filterCfg struct {
	attributeName operators.JsonReference
	operator      string
	term          string
}
*/

/*
func getFilterConfigFromSpec(c interface{}) (criteria.Criterion, error) {
	var err error
	var criterion criteria.Criterion
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

func getFilterConfigsFromSpec(c []interface{}) ([]criteria.Criterion, error) {
	filtersObj := make([]criteria.Criterion, 0)
	for _, f := range c {
		crit, err := getFilterConfigFromSpec(f)
		if err != nil {
			return nil, err
		}

		filtersObj = append(filtersObj, crit)
	}

	return filtersObj, nil
}
*/
