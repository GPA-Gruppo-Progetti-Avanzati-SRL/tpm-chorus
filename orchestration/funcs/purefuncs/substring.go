package purefuncs

import (
	"fmt"
)

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
