package operators_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/stretchr/testify/require"
	"testing"
)

type InputWanted struct {
	key      string
	withI    int
	lenArray int
}

func TestSplitKeyIdentifier(t *testing.T) {

	input := []InputWanted{
		{"doc.arr[*].pippo", -1, 4},
		{"doc.arr[i].pippo", 2, 4},
	}

	for i, iw := range input {
		res, withI, err := operators.SplitKeySpecifier(iw.key)
		require.NoError(t, err)
		require.EqualValues(t, len(res), iw.lenArray, fmt.Sprintf("%d - %s --> %s", i, iw.key, res))
		require.EqualValues(t, withI, iw.withI, fmt.Sprintf("%d - %s --> %d", i, iw.key, withI))
	}
}
