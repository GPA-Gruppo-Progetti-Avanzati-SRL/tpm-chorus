package wfcase_test

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/globals"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
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

var jsonSample = []byte(`
{
  "key": {
    "rapporto": "000987654320"
  },
  "body": {
    "_bid": "000000987654320-000000000000002-1",
    "_et": "PRENOTATA-CC",
    "change-stream": {
      "clusterTime": {
        "I": 11,
        "T": 1744134905
      },
      "documentKey": "67f562f9a3929696efa8d47a",
      "resumeToken": "8267F562F90000000B2B042C0100296E5A100491D5AB8C3E8545F0B2889DD1A2B158E6463C6F7065726174696F6E54797065003C696E736572740046646F63756D656E744B65790046645F6964006467F562F9A3929696EFA8D47A000004"
    },
    "evento": {
      "Codice_Causale": "PIG",
      "Data_Contabile": "20250325",
      "Descrizione": "PPT BIO G. SRLS 162/22  VITTORIA 492/ BIS 17/3/25",
      "Identificativo_Movimento": "000000000000002-1",
      "Importo_Movimento": "7929",
      "Numero_Rapporto": "000987654320",
      "Settoriale_Mittente": "CC",
      "Tipo_Movimento": "prenotazione"
    },
    "infoTecniche": {
      "ccid": "000000000000002__SEQNO000000000000002",
      "dataAlimentazione": "2025-03-25 13:46:03.300275000000",
      "dataOperazione": "2025-03-25 13:46:02.906888000000",
      "sourceTopic": "rhapsody-prenotate-cc-src",
      "tipoOperazione": "I"
    },
    "informazioniAddizionali": {
      "_bid": "000987654320",
      "_et": "RAP",
      "_id": "67f53e0e112b12145d230c36",
      "categoria": "1210",
      "dataApertura": "20250320",
      "dataIntestazione": "20250320",
      "filiale": "38095",
      "infoTecniche": {
        "dataOra": "20250320163910",
        "dataOraAlimentazione": "2025-03-21 00:39:13.441472000000",
        "tipoOperazione": "I"
      },
      "ndg": "09876543",
      "ndgInfo": {
        "_bid": "09876543",
        "_et": "NDG",
        "_id": "67dd4558bf8d8d94af26ec18",
        "dataCensimento": "20221213",
        "dataNascita": "19800101",
        "legati": [
          {
            "dataCensimento": "20200102",
            "ndg": "1234"
          },
          {
            "dataCensimento": "20200102",
            "ndg": "0123"
          }
        ],
        "maxDataCensimento": "20200102",
        "natura": "PF"
      },
      "servizio": "CC"
    }
  }
}
`)

func TestJsonParseSample(t *testing.T) {

	m := interface{}(nil)
	err := json.Unmarshal(jsonSample, &m)
	require.NoError(t, err)

	p, err := jsonpath.Get("$.body.evento.Numero_Rapporto", m)
	require.NoError(t, err)
	t.Log(p)

	p, err = jsonpath.Get(`$.body["change-stream"].resumeToken`, m)
	require.NoError(t, err)
	t.Log(p)

	v := float64(21000)
	s := fmt.Sprintf("%v", v)
	t.Log(s)

	v = float64(21000000)
	s = fmt.Sprintf("%v", v)
	t.Log(s)

	s = fmt.Sprintf("%.0f", v)
	t.Log(s)
}
