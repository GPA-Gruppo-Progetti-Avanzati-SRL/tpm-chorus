package scriptactivity

import (
	"context"
	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/stretchr/testify/require"
	"testing"
)

var jsonDataWithLegati = []byte(`
{
  "_id": {
    "$oid": "673a39a2a1d1cf5c9eb3f08f"
  },
  "_bid": "55325481",
  "_et": "NDG",
  "dataCensimento": "20241030",
  "dataNascita": "19800101",
  "natura": "SPA",
  "legati": [
    {
      "dataCensimento": "20241130",
      "ndg": "7046450"
    },
    {
      "dataCensimento": "20241131",
      "ndg": "7046450"
    }
  ]
}
`)

var jsonDataWithoutLegati = []byte(`
{
  "_id": {
    "$oid": "673a39a2a1d1cf5c9eb3f08f"
  },
  "_bid": "55325481",
  "_et": "NDG",
  "dataCensimento": "20241030",
  "dataNascita": "19800101",
  "natura": "SPA"
}
`)

var scriptText = `
json := import("json")
fmt := import("fmt")

each := func(seq, fn) {
    for x in seq { fn(x) }
}

obj := json.decode(jsonData)   
if is_error(obj) {
   fmt.println(obj.value)
}

computedDate := ""
each(obj.legati, func(x) {
    if x.dataCensimento > computedDate {
       computedDate = x.dataCensimento
    }
})

if computedDate == "" {
	computedDate = obj.dataCensimento
}
`

func TestScriptJsonParsing(t *testing.T) {
	script := tengo.NewScript([]byte(scriptText))
	script.SetImports(stdlib.GetModuleMap("fmt", "json"))

	_ = script.Add("jsonData", jsonDataWithLegati)
	compiled, err := script.RunContext(context.Background())
	require.NoError(t, err)

	dt := compiled.Get("computedDate")
	t.Log("computed date: ", dt)

	script = tengo.NewScript([]byte(scriptText))
	script.SetImports(stdlib.GetModuleMap("fmt", "json"))
	_ = script.Add("jsonData", jsonDataWithoutLegati)
	compiled, err = script.RunContext(context.Background())
	require.NoError(t, err)

	dt = compiled.Get("computedDate")
	t.Log("computed date: ", dt)
}
