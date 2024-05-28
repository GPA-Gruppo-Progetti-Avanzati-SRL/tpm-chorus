package operators

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strconv"
)

type conversion struct {
	targetRef     JsonReference
	convType      string
	sourceUnit    string
	targetUnit    string
	decimalFormat bool
}

func getConversionSpecw(c interface{}) (conversion, error) {

	const semLogContext = "kazaam-format::get-conversion-from-spec"
	var err error
	pcfg := conversion{}

	pcfg.targetRef, err = getJsonReferenceParamFromMap(c, SpecParamTargetReference, true)
	if err != nil {
		return pcfg, err
	}

	pcfg.convType, err = getStringParamFromMap(c, SpecParamConversionType, true)
	if err != nil {
		return pcfg, err
	}

	if pcfg.convType == "amt" {
		pcfg.sourceUnit, err = getStringParamFromMap(c, SpecParamSourceUnit, true)
		if err != nil {
			return pcfg, err
		}

		pcfg.targetUnit, err = getStringParamFromMap(c, SpecParamTargetUnit, true)
		if err != nil {
			return pcfg, err
		}

		pcfg.decimalFormat, err = getBoolParamFromMap(c, SpecParamDecimalFormat, false)
		if err != nil {
			return pcfg, err
		}
	}

	log.Info().Str(SpecParamTargetReference, pcfg.targetRef.Path).Str(SpecParamSourceUnit, pcfg.sourceUnit).Str(SpecParamTargetUnit, pcfg.targetUnit).Bool(SpecParamDecimalFormat, pcfg.decimalFormat).Msg(semLogContext)
	return pcfg, nil
}

func Format(_ kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-format::execute"
		var err error

		conversions, err := getArrayParam(spec, SpecParamConversions, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		for _, c := range conversions {

			var conv conversion
			conv, err = getConversionSpecw(c)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			var targetValue []byte
			var vt jsonparser.ValueType
			targetValue, vt, _, err = jsonparser.Get(data, conv.targetRef.Keys...)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			s := ""
			switch vt {
			case jsonparser.Number:
				fallthrough
			case jsonparser.String:
				s = string(targetValue)
			default:
				err = fmt.Errorf("format %d not supported for element %s", vt, conv.targetRef.Path)
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			log.Debug().Interface("from-value", string(targetValue)).Str("to-value", s).Interface("of-type", vt).Msg(semLogContext)

			iv := 0
			switch conv.convType {
			case "amt":
				s, err = purefuncs.Amt(purefuncs.AmountOpAdd, conv.sourceUnit, conv.targetUnit, conv.decimalFormat, s)
				if err == nil {
					data, err = jsonparser.Set(data, []byte(s), conv.targetRef.Keys...)
				}
			case "atoi":
				iv, err = strconv.Atoi(s)
				if err != nil {
					return data, err
				}
				data, err = jsonparser.Set(data, []byte(fmt.Sprint(iv)), conv.targetRef.Keys...)
			}
		}

		return data, err
	}
}
