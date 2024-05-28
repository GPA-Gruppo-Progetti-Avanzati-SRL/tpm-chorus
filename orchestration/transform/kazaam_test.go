package transform_test

import (
	_ "embed"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform/operators"
	"github.com/qntfy/kazaam"
	"github.com/stretchr/testify/require"
	"testing"
)

var coalesceRule = []byte(`
[{ "operation": "coalesce", "spec": { "objid": ["doc.pippo", "0"] } }]	
`)

var coalesceInput = `
{ "doc": { "pluto": 5 } }	
`

func TestKazaam(t *testing.T) {
	kc := kazaam.NewDefaultConfig()
	err := kc.RegisterTransform(transform.OperatorShiftArrayItems, operators.ShiftArrayItems(kc))
	require.NoError(t, err)
	err = kc.RegisterTransform(transform.OperatorFormat, operators.Format(kc))
	require.NoError(t, err)

	k, err := kazaam.New(string(coalesceRule), kc)
	require.NoError(t, err)

	kazaamOut, err := k.TransformJSONStringToString(coalesceInput)
	require.NoError(t, err)
	fmt.Println(kazaamOut)
}
