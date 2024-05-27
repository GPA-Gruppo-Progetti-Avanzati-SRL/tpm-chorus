package purefuncs

import (
	"github.com/rs/zerolog/log"
	"regexp"
)

/*
RegexMatch reports whether the string value contains any match of the regular expression pattern
*/
func RegexMatch(pattern string, value string) bool {
	const semLogContext = "orchestration-funcs::regex"

	match, err := regexp.MatchString(pattern, value)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return false
	}
	return match
}
