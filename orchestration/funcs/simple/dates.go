package simple

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

func IsDate(value interface{}, layouts ...string) bool {
	rc := false
	switch t := value.(type) {
	case time.Time:
		rc = true
	case string:
		for _, layout := range layouts {
			if _, err := time.Parse(layout, t); err == nil {
				rc = true
				break
			}
		}
	}

	return rc
}

func ParseDate(value interface{}, location string, layouts ...string) interface{} {

	const semLogContext = "orchestration-funcs::parse-date"

	var err error
	var tm time.Time

	loc, err := time.LoadLocation(location)
	if err != nil {
		log.Error().Err(err).Str("time-location", location).Msg(semLogContext)
		loc = time.UTC
	}

	switch t := value.(type) {
	case time.Time:
		return t.In(loc)
	case string:
		for _, layout := range layouts {
			if tm, err = time.ParseInLocation(layout, t, loc); err == nil {
				return tm.In(loc)
			}
		}

		log.Error().Str("date-value", t).Interface("layouts", layouts).Msg(semLogContext + " un-parsable date")
	default:
		log.Error().Str("type", fmt.Sprintf("%T", value)).Msg(semLogContext + " unrecognized type")
	}

	return nil
}

func ParseAndFmtDate(value interface{}, location string, targetLayout string, inputLayouts ...string) string {

	const semLogContext = "orchestration-funcs::parse-and-fmt-date"

	var s string

	loc, err := time.LoadLocation(location)
	if err != nil {
		log.Error().Err(err).Str("time-location", location).Msg(semLogContext)
		loc = time.UTC
	}

	switch t := value.(type) {
	case time.Time:
		s = t.In(loc).Format(targetLayout)
		return s
	case string:
		for i := range inputLayouts {
			if tm, err := time.ParseInLocation(inputLayouts[i], t, loc); err == nil {
				s = tm.In(loc).Format(targetLayout)
				break
			}
		}

		if s == "" {
			log.Error().Str("date-value", t).Interface("layouts", inputLayouts).Msg(semLogContext + " un-parsable date")
		}

	default:
		log.Error().Str("type", fmt.Sprintf("%T", value)).Msg(semLogContext + " unrecognized type")
	}

	return s
}

func DateDiff(value1, value2 interface{}, outputUnit string, inputLayouts ...string) int {

	const semLogContext = "orchestration-funcs::date-diff"

	i1 := ParseDate(value1, "Local", inputLayouts...)
	i2 := ParseDate(value2, "Local", inputLayouts...)

	const SemLogDateValue2 = "date-value-2"
	const SemLogDateValue1 = "date-value-1"
	if i1 == nil || i2 == nil {
		log.Error().Interface(SemLogDateValue1, value1).Interface(SemLogDateValue2, value2).Interface("layouts", inputLayouts).Msg(semLogContext + " un-parsable dates")
		return 0
	}

	tm1, ok1 := i1.(time.Time)
	tm2, ok2 := i2.(time.Time)

	if !ok1 || !ok2 {
		log.Error().Interface(SemLogDateValue1, value1).Interface(SemLogDateValue2, value2).Interface("layouts", inputLayouts).Msg(semLogContext + " values are not time.Time")
		return 0
	}

	diff := tm1.Sub(tm2)
	log.Trace().Interface(SemLogDateValue1, value1).Interface(SemLogDateValue2, value2).Interface("layouts", inputLayouts).Dur("diff", diff).Msg(semLogContext)

	out := 0
	switch strings.ToLower(outputUnit) {
	case "days":
		out = int(diff.Hours() / 24)
	case "hours":
		out = int(diff.Hours())
	case "minutes":
		out = int(diff.Minutes())
	case "seconds":
		out = int(diff.Seconds())
	default:
		log.Trace().Interface(SemLogDateValue1, value1).Interface(SemLogDateValue2, value2).Dur("diff", diff).Str("output-unit", outputUnit).Interface("layouts", inputLayouts).Msg(semLogContext + " unrecognized output-unit")
	}

	return out
}
