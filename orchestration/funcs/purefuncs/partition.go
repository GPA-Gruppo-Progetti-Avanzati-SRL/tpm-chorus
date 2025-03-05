package purefuncs

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/partitionutil"
)

func HashPartition(data interface{}, numPartitions int) int {
	const semLogContext = "orchestration-funcs::hash-partition"

	var b []byte
	switch param := data.(type) {
	case string:
		b = []byte(param)
	case []byte:
		b = param
	default:
		b = []byte(fmt.Sprint(data))
	}

	return partitionutil.HashPartition(b, numPartitions)
}
