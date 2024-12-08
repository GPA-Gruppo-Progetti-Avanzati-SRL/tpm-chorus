package lenarrays

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/qntfy/kazaam/transform"
)

type LenArraysMapping struct {
	src operators.JsonReference
	dst operators.JsonReference
}

type LenArraysParams struct {
	mapping []LenArraysMapping
}

func getParamsFromSpec(spec *transform.Config) (LenArraysParams, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"

	params := LenArraysParams{}
	for n, _ := range *spec.Spec {
		s, err := operators.GetStringParam(spec, n, false, "")
		if err != nil {
			return params, err
		}

		if s != "" {
			m := LenArraysMapping{src: operators.MustToJsonReference(s), dst: operators.MustToJsonReference(n)}
			params.mapping = append(params.mapping, m)
		}
	}

	return params, nil
}
