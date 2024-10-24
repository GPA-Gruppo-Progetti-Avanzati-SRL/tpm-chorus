package transformactivity_test

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/transformactivity"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSetMapProperty(t *testing.T) {

	m := map[string]interface{}{
		"lev0_1": "lev0_1_value",
		"lev0_2": "lev0_2_value",
		"lev2": map[string]interface{}{
			"lev2_1": "lev2_1_value",
			"lev2_2": "lev2_2_value",
		},
	}

	m, err := transformactivity.SetMapProperty(m, "pippo", map[string]interface{}{"new-val": "new-val-value"})
	require.NoError(t, err)
	b, err := json.Marshal(m)
	require.NoError(t, err)
	t.Log(string(b))

	m, err = transformactivity.SetMapProperty(m, "lev2.pippo2", map[string]interface{}{"new-val": "new-val-value"})
	require.NoError(t, err)
	b, err = json.Marshal(m)
	require.NoError(t, err)
	t.Log(string(b))

	m, err = transformactivity.SetMapProperty(m, "lev2bis.pippo2bis", map[string]interface{}{"new-val": "new-val-value"})
	require.NoError(t, err)
	b, err = json.Marshal(m)
	require.NoError(t, err)
	t.Log(string(b))
}
