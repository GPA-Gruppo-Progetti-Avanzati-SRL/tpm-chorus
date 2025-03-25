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

func RegexExtractFirstWithDefault(pattern string, value string, defaultVal string) string {
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

	return defaultVal
}

func RegexSetMatchAndExtract(value string, ifMatchButNoExtractionValue string, ifNoMatchValue string, patterns ...string) string {
	const semLogContext = "orchestration-funcs::regex-set-match-extract-first"

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return ""
		}

		matches := re.FindAllSubmatch([]byte(value), -1)
		log.Trace().Int("number-of-matches", len(matches)).Str("pattern", pattern).Str("value", value).Msg(semLogContext)
		for _, m := range matches {
			if len(m) > 1 {
				log.Trace().Str("value", value).Str("pattern", pattern).Str("result", string(m[1])).Msg(semLogContext + " - matched and extracted")
				return string(m[1])
			}
		}

		if len(matches) > 0 {
			log.Trace().Str("value", value).Str("pattern", pattern).Msg(semLogContext + " - matched but no extraction")
			return ifMatchButNoExtractionValue
		}
	}

	log.Warn().Str("value", value).Int("num-patterns", len(patterns)).Msg(semLogContext + " doesn't match any pattern")
	return ifNoMatchValue
}
