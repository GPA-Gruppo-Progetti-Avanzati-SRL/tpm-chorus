package operators

import (
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
)

func NoOp(_ kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = "kazaam-no-op::execute"
		var err error

		for k, n := range *spec.Spec {
			log.Info().Interface(k, n).Msg(semLogContext)
		}

		return data, err
	}

}
