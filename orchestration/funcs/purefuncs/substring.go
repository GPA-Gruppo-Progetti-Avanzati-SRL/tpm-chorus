package purefuncs

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
)

func Substr(elem interface{}, start float64, end float64) string {
	const semLogContext = "orchestration-funcs::substr"

	if elem == nil {
		return ""
	}

	s := fmt.Sprintf("%v", elem)
	istart := int(start)
	iend := int(end)
	if len(s) <= iend {
		iend = len(s)
	}

	if iend <= istart || istart < 0 || iend <= 0 {
		log.Error().Err(errors.New("")).Str("s", s).Int("start", istart).Int("end", iend).Msg(semLogContext)
		return ""
	}

	return s[istart:iend]
}

func Left(elem interface{}, length float64) string {
	s := fmt.Sprintf("%v", elem)

	l := int(length)
	if len(s) <= l {
		return s
	}

	return s[:l]
}

func Right(elem interface{}, length float64) string {
	s := fmt.Sprintf("%v", elem)

	l := int(length)
	if len(s) <= l {
		return s
	}

	return s[len(s)-l:]
}

func Len(elem interface{}) int {
	s := fmt.Sprintf("%v", elem)
	return len(s)
}
