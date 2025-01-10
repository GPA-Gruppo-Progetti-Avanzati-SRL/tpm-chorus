package addarrayitems

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/qntfy/kazaam/transform"
)

type ParamsMapping struct {
	src operators.JsonReference
	dst operators.JsonReference
}

type Params struct {
	mapping []ParamsMapping
}

func getParamsFromSpec(spec *transform.Config) (Params, error) {
	const semLogContext = "kazaam-filter-array::get-params-from-specs"

	params := Params{}
	for n, _ := range *spec.Spec {
		s, err := operators.GetStringParam(spec, n, false, "")
		if err != nil {
			return params, err
		}

		if s != "" {
			m := ParamsMapping{src: operators.MustToJsonReference(s), dst: operators.MustToJsonReference(n)}
			params.mapping = append(params.mapping, m)
		}
	}

	return params, nil
}
