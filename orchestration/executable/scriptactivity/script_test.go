package scriptactivity

import (
	"context"
	"errors"
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

var jsonPromo = []byte(`
{
  "promozioni": [
    {
      "titolo": "Start: Un amico per te",
      "descrizione": "chi trova un amico trova un tesoro",
      "sconto": "non paghi la merenda per 12 mesi",
      "scontoVal": "1",
      "scontoPeriod": 12
    },
    {
      "descrizione": "Puoi ottenere l'azzeramento del canone fino a un  massimo di 24 mesi se apri il conto entro il 31/07/2024. Lo sconto è previsto per i primi 12 mesi dall'apertura  del conto e per ulteriori 12 nei mesi in cui è presente l'accredito dello stipendio/pensione o un saldo  contabile medio mensile superiore a 2.500€. Per dettagli sulla definizione di accredito e informazioni  aggiuntive consulta il Regolamento.",
      "sconto": "€0 / mese fina a 24 mesi",
      "titolo": "PROMOZIONE  DIGITAL 2024",
      "scontoVal":0,
      "scontoPeriod": 12
    }
  ]
}
`)

var scriptCmpText = `
json := import("json")
fmt := import("fmt")

each := func(seq, fn) {
    for i, v in seq { fn(i, v) }
}

result := ""
scontoNdx := -1

obj := json.decode(jsonData)
if is_error(obj) { 
   fmt.println(obj.value)
   exit(obj)
}

minSconto := 100000000.00
each(obj.promozioni, func(i, v) {
       if float(v.scontoVal) < minSconto {
           scontoNdx = i
           minSconto = float(v.scontoVal)
       }
})

if scontoNdx >= 0 {
   result = obj.promozioni[scontoNdx].sconto
} else {
   result = ""
}
`

var ErrUserExit = errors.New("user exit")

func TestScriptCompare(t *testing.T) {
	script := tengo.NewScript([]byte(scriptCmpText))
	script.SetImports(stdlib.GetModuleMap("fmt", "json"))

	err := script.Add("exit", &tengo.UserFunction{
		Name: "exit",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) > 0 {
				return nil, errors.New(args[0].String())
			}
			return tengo.UndefinedValue, ErrUserExit
		},
	})
	require.NoError(t, err)

	//err = script.Add("atoi", &tengo.UserFunction{
	//	Name: "atoi",
	//	Value: func(args ...tengo.Object) (tengo.Object, error) {
	//		fmt.Println(args[0], args[0].String())
	//		return nil, ErrUserExit
	//	},
	//})
	//require.NoError(t, err)

	err = script.Add("jsonData", jsonPromo)
	require.NoError(t, err)

	c, err := script.Compile()
	require.NoError(t, err)
	err = c.Run()
	require.NoError(t, err)

	ndx := c.Get("result")
	t.Log("computed index: ", ndx)
}
