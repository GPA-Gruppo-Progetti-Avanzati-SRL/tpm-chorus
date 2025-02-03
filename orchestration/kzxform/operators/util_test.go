package operators_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

type InputWanted struct {
	key      string
	withI    int
	withPlus int
	lenArray int
}

func TestSplitKeyIdentifier(t *testing.T) {

	input := []InputWanted{
		{"doc.arr[*].pippo", -1, -1, 4},
		{"doc.arr[i].pippo", 2, -1, 4},
		{"doc.arr[i].pippo[+]", 2, 4, 5},
	}

	for i, iw := range input {
		res, withI, plsuNdx, err := operators.SplitKeySpecifier(iw.key)
		require.NoError(t, err)
		require.EqualValues(t, len(res), iw.lenArray, fmt.Sprintf("%d - %s --> %s", i, iw.key, res))
		require.EqualValues(t, withI, iw.withI, fmt.Sprintf("%d - %s --> %d", i, iw.key, withI))
		require.EqualValues(t, plsuNdx, iw.withPlus, fmt.Sprintf("%d - %s --> %d", i, iw.key, plsuNdx))
	}
}

var yamlExample = []byte(`
id: "default-ex-01"
rules:
  - operation: default
    spec:
      numeroSecurizzato1: "null"
      numeroSecurizzato2: null
      numeroSecurizzato3: ""
      numeroSecurizzato4:
`)

func TestYamlDeserialization(t *testing.T) {
	var m kzxform.Config
	err := yaml.Unmarshal(yamlExample, &m)
	require.NoError(t, err)
	t.Log(m)
}
