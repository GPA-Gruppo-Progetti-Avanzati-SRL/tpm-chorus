package purefuncs

import (
	"fmt"
)

func Printf(format string, elems ...interface{}) string {
	s := fmt.Sprintf(format, elems...)
	return s
}
