package wfcase_test

import (
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"github.com/stretchr/testify/require"
	"testing"
)

var j1 = []byte(`
{
  "can-ale": "APPP",
  "beneficiario": {
    "natura": "PP",
    "tipologia": "ALIAS",
    "numero": "8188602",
    "intestazione": "MARIO ROSSI"
  },
  "ordinante": {
    "natura": "DT",
    "tipologia": "ALIAS",
    "numero": "7750602",
    "codiceFiscale": "LPRSPM46H85U177S"
  },
  "operazione": {
    "divisa": "EUR",
    "importo": 20,
    "descrizione": "string",
    "tipo": "RPAU"
  },
  "additionalProperties": {
    "additional-Prop1": {},
    "additionalProp2": {},
    "additionalProp3": {}
  },
  "array": [
   {
    "item0": "hello",
    "item1": {
      "item1-1": "valore item1-1"
    }
    }
  ]
}
`)

func TestJsonPath(t *testing.T) {

	m := interface{}(nil)
	err := json.Unmarshal(j1, &m)
	require.NoError(t, err)

	v1, err := jsonpath.Get("$", m)
	require.NoError(t, err)
	t.Log(v1)

	v, err := jsonpath.Get("$[\"can-ale\"]", m)
	require.NoError(t, err)
	t.Log(v)

	v, err = jsonpath.Get("$.operazione", m)
	require.NoError(t, err)

	if _, ok := v.(map[string]interface{}); ok {
		b, err := json.Marshal(v)
		require.NoError(t, err)

		t.Log("map of", string(b))
	} else {
		t.Log(v)
	}

	v, err = jsonpath.Get("$.operazione.importo", m)
	require.NoError(t, err)
	t.Log(v)

	v, err = jsonpath.Get("$.array", m)
	require.NoError(t, err)

	if _, ok := v.([]interface{}); ok {
		b, err := json.Marshal(v)
		require.NoError(t, err)

		t.Log("array of", string(b))
	} else {
		t.Log(v)
	}

}
