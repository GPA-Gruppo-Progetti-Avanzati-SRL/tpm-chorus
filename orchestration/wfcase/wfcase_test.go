package wfcase_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/globals"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/PaesslerAG/gval"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

var j = []byte(`
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
    "importo": 10,
    "descrizione": "string",
    "tipo": "RPAU"
  },
  "additionalProperties": {
    "additionalProp1": {},
    "additionalProp2": {},
    "additionalProp3": {}
  },
  "operazioni": [{
      "errori-ope": [{
          "dsc-errore": "mio errore"
      }],
      "pippo": "pluto"
  }]
}
`)

func TestNewProcessVarResolver(t *testing.T) {

	pvs := wfexpressions.ProcessVars(make(map[string]interface{}))
	resolver, err := wfexpressions.NewEvaluator("no-name", wfexpressions.WithBody(constants.ContentTypeApplicationJson, j, ""))
	require.NoError(t, err)

	r, err := pvs.Eval("donotexist")
	require.NoError(t, err)
	t.Log(r)

	err = interpolateEvaluateAndSet(pvs, "beneficiario_natura", "{$.beneficiario.natura}", resolver, false, -1)
	require.NoError(t, err)
	err = interpolateEvaluateAndSet(pvs, "can_ale", "{$[\"can-ale\"]}", resolver, false, -1)
	require.NoError(t, err)

	err = interpolateEvaluateAndSet(pvs, "beneficiario_numero", "{$.beneficiario.numero}", resolver, false, -1)
	require.NoError(t, err)

	err = interpolateEvaluateAndSet(pvs, "beneficiario_numero2", `{$["operazioni"][0]["errori-ope"][0]["dsc-errore"]}`, resolver, false, -1)
	require.NoError(t, err)

	err = interpolateEvaluateAndSet(pvs, "beneficiario_numero3", "{$.operazioni[0].pippo}", resolver, false, -1)
	require.NoError(t, err)

	err = interpolateEvaluateAndSet(pvs, "operazione_importo", "{$.operazione.importo}", resolver, false, -1)
	require.NoError(t, err)

	t.Log(pvs)

	ndx, err := pvs.IndexOfTheOnlyOneTrueExpression([]string{`beneficiario_natura == "DT"`, `beneficiario_numero == "8188602"`})
	require.NoError(t, err)
	require.Equal(t, ndx, 1)

	pvs["map"] = func(s ...string) string {
		return strings.Join(s, " ")
	}

	ndx, err = pvs.IndexOfTheOnlyOneTrueExpression([]string{`map("hello", "world!") == "hello world!"`})
	require.NoError(t, err)
	require.Equal(t, ndx, 0)
}

type GValEvaluator struct {
	Vars wfexpressions.ProcessVars
}

func TestGVal(t *testing.T) {

	pvs := wfexpressions.ProcessVars(make(map[string]interface{}))
	resolver, err := wfexpressions.NewEvaluator("no-name", wfexpressions.WithBody(constants.ContentTypeApplicationJson, j, ""))
	require.NoError(t, err)

	err = interpolateEvaluateAndSet(pvs, "beneficiario_natura", "{$.beneficiario.natura}", resolver, false, -1)
	require.NoError(t, err)
	err = interpolateEvaluateAndSet(pvs, "beneficiario_numero", "{$.beneficiario.numero}", resolver, false, -1)
	require.NoError(t, err)

	t.Log(pvs)

	evalMap := make(map[string]interface{})
	evalMap["v"] = pvs

	exprValue, err := gval.Evaluate(`v.beneficiario_natura == "DT"`, evalMap)
	require.NoError(t, err)
	require.Equal(t, exprValue, false)

	exprValue, err = gval.Evaluate(`beneficiario_natura2`, evalMap)
	require.NoError(t, err)

	evalMap["dict"] = func(s ...string) string {
		return strings.Join(s, " ")
	}

	exprValue, err = gval.Evaluate(`dict(v.beneficiario_natura, "Beneficiario")`, evalMap)
	require.NoError(t, err)
	require.Equal(t, exprValue, "PP Beneficiario")

	exprValue, err = gval.Evaluate(`"hello"`, evalMap)
	require.NoError(t, err)
	require.Equal(t, "hello", exprValue)
}

func interpolateEvaluateAndSet(pvs map[string]interface{}, n string, expr string, resolver *wfexpressions.Evaluator, globalScope bool, ttl time.Duration) error {

	val, _, err := varResolver.ResolveVariables(expr, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return err
	}

	val, isExpr := wfcase.IsExpression(val)

	// Was isExpression(val) but in doing this I use the evaluated value and I depend on the value of the variables  with potentially weird values.
	var varValue interface{} = val
	if isExpr && val != "" {
		varValue, err = gval.Evaluate(val, pvs)
		if err != nil {
			return err
		}
	}

	if globalScope {
		err = globals.SetGlobalVar("", n, varValue, ttl)
	} else {
		pvs[n] = varValue
	}

	return nil
}
