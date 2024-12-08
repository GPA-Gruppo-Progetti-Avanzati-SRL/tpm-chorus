package distinctarrayitems

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	SpecParamSourceReference = "source-ref"
	SpecParamTargetReference = "target-ref"
	SpecParamDistinctOn      = "on"
)

type DistinctArrayItemsParams struct {
	sourceRef operators.JsonReference
	destRef   operators.JsonReference
	On        operators.JsonReference
}

func getDistinctArrayItemsParamsFromSpec(spec *transform.Config) (DistinctArrayItemsParams, error) {
	const semLogContext = "kazaam-distinct-array-items::get-params-from-specs"
	var err error

	params := DistinctArrayItemsParams{}

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

	params.On, err = operators.GetJsonReferenceParam(spec, SpecParamDistinctOn, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return params, err
	}

	return params, nil
}
