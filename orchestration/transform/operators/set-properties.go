package operators

import (
	"encoding/json"

	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

type PropertyConfig struct {
	Name      JsonReference
	Value     []byte
	IfMissing bool
}

func getPropertyConfigFromSpec(c interface{}) (PropertyConfig, error) {
	var err error
	pcfg := PropertyConfig{}

	pcfg.Name, err = getJsonReferenceParamFromMap(c, SpecParamPropertyNameRef, true)
	if err != nil {
		return pcfg, err
	}

	pv, err := getParamFromMap(c, SpecParamPropertyValue, true)
	if err != nil {
		return pcfg, err
	}

	pcfg.Value, err = json.Marshal(pv)
	if err != nil {
		return pcfg, err
	}

	/*
		switch pvt := pv.(type) {
		case string:
			pcfg.Value, err = json.Marshal(pvt)
		case int:
			pcfg.Value = []byte(fmt.Sprint(pvt))
		default:
			pcfg.Value, err = json.Marshal(pvt)
			if err != nil {
				return pcfg, err
			}
		}
	*/

	pcfg.IfMissing, err = getBoolParamFromMap(c, SpecParamIfMissing, false)
	if err != nil {
		return pcfg, err
	}

	return pcfg, nil
}

func SetProperties(_ kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-set-property::execute"
		var err error

		props, err := getArrayParam(spec, SpecParamProperties, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		for _, c := range props {
			pcfg, err := getPropertyConfigFromSpec(c)
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
