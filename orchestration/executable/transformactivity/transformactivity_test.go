package transformactivity_test

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/transformactivity"
	"github.com/PaesslerAG/jsonpath"
	"github.com/qntfy/jsonparser"
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

var SourceData = []byte(`{"_bid":"000987654320-2-1","_et":"PRENOTATA-CC","evento":{"Codice_Causale":"PIG","Data_Contabile":"20250325","Descrizione":"PPT BIO G. SRLS 162/22 VITTORIA 492/ BIS 17/3/25","Identificativo_Movimento":"2-1","Importo_Movimento":"7929","Numero_Rapporto":"000987654320","Settoriale_Mittente":"CC","Tipo_Movimento":"prenotazione"},"infoTecniche":{"ccid":"3482442933092810753_-2259779848042971135","dataAlimentazione":"2025-03-25 13:46:03.300275000000","dataOperazione":"2025-03-25 13:46:02.906888000000","sourceTopic":"rhapsody-prenotate-cc-src","tipoOperazione":"I"}}`)
var AddData = []byte(`{"_id":"67cdb1a51e1a42eabfc8c67e","servizio":"CC","_et":"RAP","_bid":"000059356949","categoria":"1210","filiale":"10168","dataApertura":"20041120","dataIntestazione":"20041120","infoTecniche":{"audSeqno":"16494249","dataOra":"2025-03-09 16:19:46","dataOraAlimentazione":"2025-03-09 02:25:43","job":"Batch_TRAP","program":"ETL","tipoOperazione":"UPSERT"},"ndg":"28684203","ndgInfo":{"dataCensimento":"20041120","natura":"COI","ndg":"28684203","legati":[{"ndg":"5624209", "dataCensimento":"20041120"},{"dataCensimento":"20041120","ndg":"13832508"}]}}`)

const TargetProperty = "InformazioniAddizionali"

func TestMerge(t *testing.T) {
	reference := setProperty(t, TargetProperty, SourceData, AddData)
	t.Log(string(reference))

	for i := 1; i < 100000; i++ {
		b := setProperty(t, TargetProperty, SourceData, AddData)
		require.True(t, testEq(reference, b), "[%d] - mapping is different", i)
	}

}

func BenchmarkJsonPath(b *testing.B) {
	for b.Loop() {
		var m interface{}
		err := json.Unmarshal(SourceData, &m)
		if err != nil {
			panic(err)
		}

		for i := 0; i < 10; i++ {
			_, err = jsonpath.Get("$.infoTecniche.dataOperazione", m)
			if err != nil {
				panic(err)
			}
		}
	}
}

func BenchmarkKazaamParser(b *testing.B) {
	for b.Loop() {
		for i := 0; i < 10; i++ {
			_, _, _, err := jsonparser.Get(SourceData, "infoTecniche", "dataOperazione")
			if err != nil {
				panic(err)
			}
		}
	}
}

func BenchmarkPrintf1(b *testing.B) {
	for b.Loop() {
		s1 := "s1"
		s2 := "s2"
		s3 := "s3"
		_ = fmt.Sprintf("%s, %s, %s", s1, s2, s3)
	}
}

func BenchmarkPrintf2(b *testing.B) {
	for b.Loop() {
		_ = fmt.Sprintf("%s, %s, %s", funcReturningStringArray()...)
	}
}

func funcReturningStringArray() []any {
	return []any{"s1", "s2", "s3"}
}

func setProperty(t *testing.T, propName string, source, add []byte) []byte {
	var m map[string]interface{}
	err := json.Unmarshal(source, &m)
	require.NoError(t, err)

	var temp map[string]interface{}
	err = json.Unmarshal(add, &temp)
	require.NoError(t, err)
	m, err = transformactivity.SetMapProperty(m, propName, temp)
	require.NoError(t, err)

	b, err := json.Marshal(m)
	require.NoError(t, err)

	return b
}

func testEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
