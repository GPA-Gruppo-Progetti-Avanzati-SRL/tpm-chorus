package withenvfuncs

import (
	"github.com/rs/zerolog/log"
	"tpm-chorus/orchestration/config"
)

func Dict(dicts config.Dictionaries, dictName string, elems ...string) string {
	const semLogContext = "orchestration-funcs::dict"
	s, err := dicts.Map(dictName, elems...)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}
	return s
}
