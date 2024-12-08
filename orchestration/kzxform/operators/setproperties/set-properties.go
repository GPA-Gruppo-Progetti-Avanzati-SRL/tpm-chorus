package setproperties

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"

	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

const (
	OperatorSetProperties = "set-properties"
	OperatorSemLogContext = "set-property"
)

func SetProperties(_ kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"
		var err error

		props, err := operators.GetArrayParam(spec, SpecParamProperties, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		for _, c := range props {
			pcfg, err := getParamsFromSpec(c)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}

			if pcfg.IfMissing {
				_, vt, _, _ := jsonparser.Get(data, pcfg.Name.Keys...)
				if vt == jsonparser.NotExist || vt == jsonparser.Null {
					data, err = jsonparser.Set(data, pcfg.Value, pcfg.Name.Keys...)
					if err != nil {
						log.Error().Err(err).Msg(semLogContext)
						return nil, err
					}
				}
			} else {
				data, err = jsonparser.Set(data, pcfg.Value, pcfg.Name.Keys...)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return nil, err
				}
			}
		}

		return data, err
	}
}
