package format

import (
	"fmt"
	"strconv"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs/amt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	OperatorFormat        = "format"
	OperatorSemLogContext = OperatorFormat
)

func Format(_ kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"
		var err error

		conversions, err := operators.GetArrayParam(spec, SpecParamConversions, true)
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
				s, err = amt.Amt(amt.AmountOpAdd, conv.sourceUnit, conv.targetUnit, conv.decimalFormat, s)
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
