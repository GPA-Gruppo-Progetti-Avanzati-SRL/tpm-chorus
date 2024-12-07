package operators

import (
	"fmt"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

type LenArraysMapping struct {
	src JsonReference
	dst JsonReference
}

type LenArraysParams struct {
	mapping []LenArraysMapping
}

func getLenArraysParamsFromSpec(spec *transform.Config) (LenArraysParams, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"

	params := LenArraysParams{}
	for n, _ := range *spec.Spec {
		s, err := getStringParam(spec, n, false, "")
		if err != nil {
			return params, err
		}

		if s != "" {
			m := LenArraysMapping{src: MustToJsonReference(s), dst: MustToJsonReference(n)}
			params.mapping = append(params.mapping, m)
		}
	}

	return params, nil
}

func LenArrays(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-len-arrays::execute"

		params, err := getLenArraysParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		if len(params.mapping) == 0 {
			return data, nil
		}

		// clone the data... in place process has some glitches.
		outData := make([]byte, len(data))
		copy(outData, data)

		for _, m := range params.mapping {
			if m.src.WithArrayISpecifierIndex < 0 {
				outData, err = computeAndSetLenOfArray(outData, m.dst, data, m.src)
			} else {
				outData, err = computeAndSetLenOfNestedArray(outData, m.dst, data, m.src)
			}
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		}

		return outData, err
	}
}

func computeAndSetLenOfArray(outData []byte, dst JsonReference, data []byte, src JsonReference) ([]byte, error) {
	const semLogContext = "kazaam-len-arrays::compute-len"

	sourceArray, err := getJsonArray(data, src)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return outData, err
	}

	var loopNdx int
	_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		loopNdx++
	})

	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return outData, err
	}

	outData, err = jsonparser.Set(outData, []byte(fmt.Sprintf("%d", loopNdx)), dst.Keys...)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return outData, err
	}

	return outData, nil
}

func computeAndSetLenOfNestedArray(outData []byte, dst JsonReference, data []byte, src JsonReference) ([]byte, error) {
	const semLogContext = "kazaam-len-arrays::compute-nested-len"

	rootRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     src.Path[:strings.Index(src.Path, "[i]")],
		Keys:                     src.Keys[:src.WithArrayISpecifierIndex],
	}

	rootArray, err := getJsonArray(data, rootRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	var loopErr error
	var loopIndex int
	_, err = jsonparser.ArrayEach(rootArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		nestedRef := JsonReference{
			WithArrayISpecifierIndex: -1,
			Path:                     strings.ReplaceAll(src.Path[strings.Index(src.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", loopIndex)),
			// Keys:                     make([]string, len(sourceRef.Keys)),
		}
		nestedRef.Keys = append(nestedRef.Keys, src.Keys[src.WithArrayISpecifierIndex:]...)
		nestedRef.Keys[0] = fmt.Sprintf("[%d]", loopIndex)

		nestedLen, err := lenOfArray(rootArray, nestedRef)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		var nestedTargetRefKeys []string
		if dst.WithArrayISpecifierIndex >= 0 {
			nestedTargetRefKeys = append(nestedTargetRefKeys, dst.Keys...)
			nestedTargetRefKeys[dst.WithArrayISpecifierIndex] = fmt.Sprintf("[%d]", loopIndex)
		} else {
			nestedTargetRefKeys = append(nestedTargetRefKeys, dst.Keys...)
		}
		outData, err = jsonparser.Set(outData, []byte(fmt.Sprintf("%d", nestedLen)), nestedTargetRefKeys...)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		loopIndex++
	})

	return outData, nil
}
