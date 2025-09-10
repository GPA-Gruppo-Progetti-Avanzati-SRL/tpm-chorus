package lenarrays

import (
	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/kazaam/transform"
)

type LenArraysMapping struct {
	src operators2.JsonReference
	dst operators2.JsonReference
}

type LenArraysParams struct {
	mapping []LenArraysMapping
}

func getParamsFromSpec(spec *transform.Config) (LenArraysParams, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"

	params := LenArraysParams{}
	for n, _ := range *spec.Spec {
		s, err := operators2.GetStringParam(spec, n, false, "")
		if err != nil {
			return params, err
		}

		if s != "" {
			m := LenArraysMapping{src: operators2.MustToJsonReference(s), dst: operators2.MustToJsonReference(n)}
			params.mapping = append(params.mapping, m)
		}
	}

	return params, nil
}
