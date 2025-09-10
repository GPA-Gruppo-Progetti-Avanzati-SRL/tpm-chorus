package mergearrays

import (
	operators2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/kz/operators"
	"github.com/qntfy/kazaam/transform"
)

type ParamsMapping struct {
	src operators2.JsonReference
	dst operators2.JsonReference
}

type Params struct {
	mapping []ParamsMapping
}

func getParamsFromSpec(spec *transform.Config) (Params, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"

	params := Params{}
	for n, _ := range *spec.Spec {
		s, err := operators2.GetStringParam(spec, n, false, "")
		if err != nil {
			return params, err
		}

		if s != "" {
			m := ParamsMapping{src: operators2.MustToJsonReference(s), dst: operators2.MustToJsonReference(n)}
			params.mapping = append(params.mapping, m)
		}
	}

	return params, nil
}
