package purefuncs

import (
	"fmt"
	"strings"
)

func PadLeft(elem interface{}, maxLength float64, padChar string) string {
	s := fmt.Sprintf("%v", elem)

	ml := int(maxLength)
	if len(s) >= ml {
		return s
	}

	padding := strings.Repeat(padChar, ml-len(s))
	return padding + s
}
