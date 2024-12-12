package setproperties

import (
	"errors"
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

			if pcfg.Name.WithArrayISpecifierIndex < 0 {
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
			} else {
				data, err = processWithIotaSpecifier(data, pcfg)
			}
		}

		return data, err
	}
}

func processWithIotaSpecifier(data []byte, params OperatorParams) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process-iota"

	rootRef := params.Name.JsonReferenceToArrayWithIotaSpecifier()
	rootArray, err := operators.GetJsonArray(data, rootRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	val := params.Value
	if !params.Path.IsZero() {
		var vt jsonparser.ValueType
		val, vt, _, _ = jsonparser.Get(data, params.Path.Keys...)
		if vt == jsonparser.NotExist || vt == jsonparser.Null {
			err = errors.New("the source path does not exists")
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
	}

	// clone the data... in place process has some glitches.
	outData := make([]byte, len(data))
	copy(outData, data)

	var loopErr error
	var loopIndex int
	_, err = jsonparser.ArrayEach(rootArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		itemRef := params.Name.JsonReferenceToArrayItemWithIotaSpecifier(loopIndex)
		_, vt, _, _ := jsonparser.Get(data, itemRef.Keys...)
		if vt == jsonparser.NotExist || vt == jsonparser.Null {
			if params.IfMissing {
				outData, err = jsonparser.Set(outData, val, itemRef.Keys...)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					loopErr = err
					return
				}
			}
		} else {
			if vt == jsonparser.Array {
				itemRef.Keys = append(itemRef.Keys, "[+]")
				outData, err = jsonparser.Set(outData, val, itemRef.Keys...)
			} else {
				outData, err = jsonparser.Set(outData, val, itemRef.Keys...)
			}
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				loopErr = err
				return
			}
		}

		// nestedRef := params.Name.JsonReferenceToArrayNestedItemWithIotaSpecifier(loopIndex)

		loopIndex++
	})

	return outData, loopErr
}
