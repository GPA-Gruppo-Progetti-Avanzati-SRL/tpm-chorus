package format

import (
	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/rs/zerolog/log"
)

const (
	SpecParamTargetReference = "target-ref"
	SpecParamConversionType  = "type"
	SpecParamSourceUnit      = "source-unit"
	SpecParamTargetUnit      = "target-unit"
	SpecParamDecimalFormat   = "decimal-format"
	SpecParamConversions     = "conversions"
)

type conversion struct {
	targetRef     operators2.JsonReference
	convType      string
	sourceUnit    string
	targetUnit    string
	decimalFormat bool
}

func getConversionSpecw(c interface{}) (conversion, error) {

	const semLogContext = "kazaam-format::get-conversion-from-spec"
	var err error
	pcfg := conversion{}

	pcfg.targetRef, err = operators2.GetJsonReferenceParamFromMap(c, SpecParamTargetReference, true)
	if err != nil {
		return pcfg, err
	}

	pcfg.convType, err = operators2.GetStringParamFromMap(c, SpecParamConversionType, true)
	if err != nil {
		return pcfg, err
	}

	if pcfg.convType == "amt" {
		pcfg.sourceUnit, err = operators2.GetStringParamFromMap(c, SpecParamSourceUnit, true)
		if err != nil {
			return pcfg, err
		}

		pcfg.targetUnit, err = operators2.GetStringParamFromMap(c, SpecParamTargetUnit, true)
		if err != nil {
			return pcfg, err
		}

		pcfg.decimalFormat, _, err = operators2.GetBoolParamFromMap(c, SpecParamDecimalFormat, false)
		if err != nil {
			return pcfg, err
		}
	}

	log.Info().Str(SpecParamTargetReference, pcfg.targetRef.Path).Str(SpecParamSourceUnit, pcfg.sourceUnit).Str(SpecParamTargetUnit, pcfg.targetUnit).Bool(SpecParamDecimalFormat, pcfg.decimalFormat).Msg(semLogContext)
	return pcfg, nil
}
