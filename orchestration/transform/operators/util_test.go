package operators_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform/operators"
	"github.com/stretchr/testify/require"
	"testing"
)

type InputWanted struct {
	key      string
	lenArray int
}

func TestSplitKeyIdentifier(t *testing.T) {

	input := []InputWanted{
		{"doc.arr[*].pippo", 2},
	}

	for i, iw := range input {
		res, err := operators.SplitKeySpecifier(iw.key)
		require.NoError(t, err)
		require.EqualValues(t, len(res), iw.lenArray, fmt.Sprintf("%d - %s --> %s", i, iw.key, res))
	}
}
