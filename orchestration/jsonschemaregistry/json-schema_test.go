package jsonschemaregistry_test

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/jsonschemaregistry"
	"github.com/stretchr/testify/require"
	"testing"
)

var schemaTest = []byte(`
{
  "$schema": "http://json-schema.org/schema#",
  "$id": "https://tpm-rhapsody/movie.schema.json",
  "title": "Movie",
  "description": "A movie in the catalog",
  "type": "object",
  "properties": {
    "codiceServizio": {
      "description": "Codice servizio",
      "type": "string",
      "enum": ["CC"]
    },
    "codiceOperazione": {
      "description": "Codice operazione",
      "type": "string",
      "enum": ["AP","AF","MI","VO","AR"]
    },
    "cointestazione": {
      "description": "Conto contestato o meno, a firma disgiunta o congiunta",
      "type": "string",
      "enum": ["NO","SID","SIC"]
    },
    "convPrimoIntestatario": {
      "description": "Il primo intestatario richiede la convenzione AGCOM",
      "type": "string",
      "enum": ["NO","SI"]
    },
    "convSecondoIntestatario": {
      "description": "Il secondo intestatario richiede la convenzione AGCOM",
      "type": "string",
      "enum": ["NO","SI"]
    },
    "codFiscPrimoIntestatario": {
       "description": "Il codice fiscale del primo intestatrio deve essere lungo 16",
       "type": "string",
       "minLength": 16,
       "maxLength": 16
    },
    "nomeListinoArricchimento": {
      "description": "Nome listino arricchimento",
      "type": "string",
      "pattern": "\\S",
      "maxLength": 20
    }
  },
  "required"  : [ "codiceServizio", "codiceOperazione", "cointestazione", "nomeListinoArricchimento" ]
}
`)

var testData = []byte(`
{
  "codiceServizio": "CC",
  "codiceOperazione": "AP",
  "tutoreAdmin": "TU",
  "cointestazione": "NO",
  "creditCard": "SI",
  "bnfPostApp": "NO",
  "ndgPrimoIntestatario": "000055306700",
  "codFiscPrimoIntestatario": "NFEGLN82A41H501Z",
  "dataNascPrimoIntestatario": "2000-01-01",
  "convPrimoIntestatario": "NO",
  "ciaePrimoIntestatario": "6075",
  "privatePrimoIntestatario": "NO",
  "dipendentePT": "NO",
  "dipendenteTerzi": "NO",
  "giovane": "SI", 
  "possessoConto": "SI/NO",
  "convPrimoIntestatrio": "SI/NO",
  "nomeListinoArricchimento": "    "
}
`)

func TestJsonSchemaError(t *testing.T) {
	err := jsonschemaregistry.Register("test", "validation-schema.json", schemaTest)
	require.NoError(t, err)

	err = jsonschemaregistry.Validate("test", "validation-schema.json", testData)
	if err != nil {
		t.Log(err)
		var schemaErr jsonschemaregistry.SchemaError
		if errors.As(err, &schemaErr) {
			fmt.Println(schemaErr)
		}
	}
}

// Se SchemaPrt == #/required --> missing properties
// InstancePrt == #/codiceServizio SchemaPrt = #/properties/codiceServizio/enum
