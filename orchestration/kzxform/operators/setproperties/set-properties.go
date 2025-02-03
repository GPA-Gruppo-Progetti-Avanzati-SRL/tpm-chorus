package setproperties

import (
	"errors"
	"fmt"
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

				ok, _, err := shouldBeSet(data, pcfg.Name.Keys, pcfg.IfMissing, pcfg.criterion)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return nil, err
				}

				if ok {
					var valVt jsonparser.ValueType
					var val []byte
					val, valVt, err = propertyValue(data, &pcfg)
					if err != nil {
						log.Error().Err(err).Str("vt", valVt.String()).Str("val", string(val)).Msg(semLogContext)
						return nil, err
					}

					data, err = jsonparser.Set(data, val, pcfg.Name.Keys...)
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

func propertyValue(data []byte, pcfg *OperatorParams) ([]byte, jsonparser.ValueType, error) {
	const semLogContext = OperatorSemLogContext + "::value"
	var err error

	var val []byte
	vt := jsonparser.NotExist

	switch {
	case pcfg.Value != nil:
		val = pcfg.Value
		vt = jsonparser.String

	case !pcfg.Path.IsZero():
		val, vt, _, _ = jsonparser.Get(data, pcfg.Path.Keys...)
		switch vt {
		case jsonparser.String:
			val = []byte(fmt.Sprintf("\"%s\"", val))
		case jsonparser.NotExist:
			fallthrough
		case jsonparser.Null:
			err = errors.New("the source path does not exists")
			log.Error().Err(err).Str("path", pcfg.Path.Path).Msg(semLogContext)
		default:
		}

	case !pcfg.Expression.IsZero():
		var res interface{}
		res, err = pcfg.Expression.Eval(data, nil)
		if err == nil {
			val = []byte(fmt.Sprintf("%v", res))
			switch res.(type) {
			case string:
				val = []byte(fmt.Sprintf("\"%s\"", val))
				vt = jsonparser.String
			case bool:
				vt = jsonparser.Boolean
			case int:
				vt = jsonparser.Number
			case float64:
				vt = jsonparser.Number
			default:
				vt = jsonparser.Unknown
			}
		} else {
			log.Error().Err(err).Str("path", pcfg.Expression.String()).Msg(semLogContext)
		}

	default:
		val = []byte("null") // pcfg.Value
		vt = jsonparser.Null
	}

	return val, vt, err
}

func shouldBeSet(data []byte, keys []string, ifMissing bool, criterion operators.Criterion) (bool, jsonparser.ValueType, error) {
	const semLogContext = OperatorSemLogContext + "::should-be-set"
	var err error
	itShould := false
	_, vt, _, _ := jsonparser.Get(data, keys...)
	if ifMissing {
		if vt == jsonparser.NotExist || vt == jsonparser.Null {
			itShould = true
		}
	} else {
		itShould = true
	}

	if !itShould {
		return itShould, vt, nil
	}

	if !criterion.IsZero() {
		itShould, err = criterion.IsAccepted(data, nil)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return false, vt, err
		}
	}

	return itShould, vt, err
}

func processWithIotaSpecifier(data []byte, params OperatorParams) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process-iota"

	rootRef := params.Name.JsonReferenceToArrayWithIotaSpecifier()
	rootArray, err := operators.GetJsonArray(data, rootRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
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

		currentItemRef := params.Name.JsonReferenceToArrayNestedItemWithIotaSpecifier(loopIndex)
		//
		ok, vt, err := shouldBeSet(value, currentItemRef.Keys, params.IfMissing, params.criterion)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		if ok {
			var val []byte
			var valVt jsonparser.ValueType
			dataObject := data
			if (!params.Path.IsZero() && params.Path.IsPathRelative) || !params.Expression.IsZero() {
				dataObject = value
			}

			val, valVt, err = propertyValue(dataObject, &params)
			if err != nil {
				log.Error().Err(err).Str("vt", valVt.String()).Str("val", string(val)).Msg(semLogContext)
				loopErr = err
				return
			}

			/*
				data, err = jsonparser.Set(data, val, params.Name.Keys...)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					loopErr = err
					return
				}
			*/

			indexedItemRef := params.Name.JsonReferenceToArrayItemWithIotaSpecifier(loopIndex)
			if vt == jsonparser.NotExist || vt == jsonparser.Null {
				outData, err = jsonparser.Set(outData, val, indexedItemRef.Keys...)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					loopErr = err
					return
				}
			} else {
				if vt == jsonparser.Array {
					indexedItemRef.Keys = append(indexedItemRef.Keys, "[+]")
					outData, err = jsonparser.Set(outData, val, indexedItemRef.Keys...)
				} else {
					outData, err = jsonparser.Set(outData, val, indexedItemRef.Keys...)
				}
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					loopErr = err
					return
				}
			}
		}

		// nestedRef := params.Name.JsonReferenceToArrayNestedItemWithIotaSpecifier(loopIndex)

		loopIndex++
	})

	return outData, loopErr
}
