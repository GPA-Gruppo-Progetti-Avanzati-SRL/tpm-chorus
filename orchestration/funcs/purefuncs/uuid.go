package purefuncs

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

/*
Uuid() returns a Random (Version 4) UUID in form of xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
In case of error returns ""
*/
func Uuid() string {

	const semLogContext = "orchestration-funcs::uuid"

	//uuid.New() potrebbe generare un panic, recupero un NewRandom e verifico l'err
	uuidValue, err := uuid.NewRandom()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return ""
	}
	return uuidValue.String()
}
