package purefuncs

import (
	"github.com/rs/zerolog/log"
	"regexp"
)

/*
RegexMatch reports whether the string value contains any match of the regular expression pattern
*/
func RegexMatch(pattern string, value string) bool {
	const semLogContext = "orchestration-funcs::regex-match"

	match, err := regexp.MatchString(pattern, value)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return false
	}
	return match
}

func RegexExtractFirst(pattern string, value string) string {
	const semLogContext = "orchestration-funcs::regex-extract-first"

	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return ""
	}

	matches := re.FindAllSubmatch([]byte(value), -1)
	log.Trace().Int("number-of-matches", len(matches)).Str("pattern", pattern).Str("value", value).Msg(semLogContext)
	for _, m := range matches {
		return string(m[1])
	}

	return ""
}
