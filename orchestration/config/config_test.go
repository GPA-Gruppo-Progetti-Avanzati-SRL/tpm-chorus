package config_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

const (
	ResponseActivityName = "end-name"
	RequestActivityName  = "start-name"

	InfoCartaBeneficiarioDTActivityName = "info-carta-beneficiario-DT"
	InfoCartaBeneficiarioDTEndpointId   = "infocarta-1"
	InfoCartaBeneficiarioDTEndpointName = "info-carta"

	VerificaIntestazioneBeneficiarioActivityName = "verifica-intestazione-beneficiario"
	VerificaIntestazioneBeneficiarioEndpointId   = "verifica-intestazione-1"
	VerificaIntestazioneBeneficiarioEndpointName = "verifica-intestazione"

	InfoContoBeneficiarioPPActivityName = "info-conto-beneficiario-PP"

	OrchestrationYAMLFileName = "tpm-symphony-orchestration.yml"
)

var cfgOrc config.Orchestration

func SetUpOrchestration(t *testing.T) {

	sa := config.NewRequestActivity().WithName(RequestActivityName)
	procVars := []config.ProcessVar{
		{
			Name:  "beneficiario_numero",
			Value: "{$.beneficiario.numero}",
			Type:  "string",
		},
		{
			Name:  "beneficiario_natura",
			Value: "{$.beneficiario.natura}",
			Type:  "string",
		},
		{
			Name:  "beneficiario_intestazione",
			Value: "{$.beneficiario.intestazione}",
			Type:  "string",
		},
	}
	sa.ProcessVars = procVars
	sa.Validations = []config.RequestValidation{
		{Expr: "beneficiario_numero == '8188602'"},
	}

	infoCartaBeneficiario := config.NewEndpointActivity().WithName(InfoCartaBeneficiarioDTActivityName)
	infoCartaBeneficiario.Endpoints = []config.Endpoint{
		{
			Id:         InfoCartaBeneficiarioDTEndpointId,
			Name:       InfoCartaBeneficiarioDTEndpointName,
			Definition: strings.Join([]string{InfoCartaBeneficiarioDTEndpointId, "yml"}, "."),
			//ProcessVars: []config.ProcessVar{
			//	{
			//		Name: "beneficiario.intestazioneSystem",
			//		Value: "{$.informazioni.titolare.intestazione}",
			//	},
			//	{
			//		Name: "beneficiario.codiceFiscaleSystem",
			//		Value: "{$.informazioni.titolare.codiceFiscale}",
			//	},
			//	{
			//		Name: "beneficiario.partitaIva",
			//		Value: "{$.informazioni.titolare.partitaIva}",
			//	},
			//	{
			//		Name: "beneficiario.tipoClientela",
			//		Value: "{$.tipologia.tipoClientela}",
			//	},
			//	{
			//		Name: "beneficiario.contoAssociato.servizio",
			//		Value: "{$.datiFinanziari.contoAssociato.servizio}",
			//	},
			//	{
			//		Name: "beneficiario.contoAssociato.filiale",
			//		Value: "{$.datiFinanziari.contoAssociato.filiale}",
			//	},
			//	{
			//		Name: "beneficiario.contoAssociato.numero",
			//		Value: "{$.datiFinanziari.contoAssociato.numero}",
			//	},
			//	{
			//		Name: "beneficiario.contoAssociato.categoria",
			//		Value: "{$.datiFinanziari.contoAssociato.categoria}",
			//	},
			//
			//	{
			//		Name: "ordinante.codiceFiscaleSystem",
			//		Value: "{$.informazioni.titolare.codiceFiscale}",
			//	},
			//	{
			//		Name: "ordinante.partitaIva",
			//		Value: "{$.informazioni.titolare.partitaIva}",
			//	},
			//	{
			//		Name: "ordinante.intestazioneSystem",
			//		Value: "{$.informazioni.titolare.intestazione}",
			//	},
			//	{
			//		Name: "ordinante.tipoClientela",
			//		Value: "{$.tipologia.tipoClientela}",
			//	},
			//	{
			//		Name: "ordinante.contoAssociato.servizio",
			//		Value: "{$.datiFinanziari.contoAssociato.servizio}",
			//	},
			//	{
			//		Name: "ordinante.contoAssociato.filiale",
			//		Value: "{$.datiFinanziari.contoAssociato.filiale}",
			//	},
			//	{
			//		Name: "ordinante.contoAssociato.numero",
			//		Value: "{$.datiFinanziari.contoAssociato.numero}",
			//	},
			//	{
			//		Name: "ordinante.contoAssociato.categoria",
			//		Value: "{$.datiFinanziari.contoAssociato.categoria}",
			//	},
			//},
			PII: config.PersonallyIdentifiableInformation{
				Domain:    "domain-infoCartaBeneficiario",
				AppliesTo: "none",
			},
		},
	}

	infoContoBeneficiario := config.NewEchoActivity().WithName(InfoContoBeneficiarioPPActivityName)
	infoContoBeneficiario.Message = "is PP"

	verificaIntestazioneBeneficiario := config.NewEndpointActivity().WithName(VerificaIntestazioneBeneficiarioActivityName)
	verificaIntestazioneBeneficiario.Endpoints = []config.Endpoint{
		{
			Id:         VerificaIntestazioneBeneficiarioEndpointId,
			Name:       VerificaIntestazioneBeneficiarioEndpointName,
			Definition: strings.Join([]string{VerificaIntestazioneBeneficiarioEndpointId, "yml"}, "."),
		},
	}

	ea2 := config.NewResponseActivity().WithName(ResponseActivityName)
	ea2.Responses = []config.Response{{RefSimpleResponse: "responseSimple.tmpl"}}

	cfgOrc = config.Orchestration{
		Id: "bpap-verifica",
		Activities: []config.Configurable{
			sa, infoCartaBeneficiario, infoContoBeneficiario, verificaIntestazioneBeneficiario, ea2,
		},
		References: config.DataReferences{
			{Path: "responseSimple.tmpl", Data: []byte(`{"msg":"hello-world"}`)},
		},
	}

	err := cfgOrc.AddPath(sa.Name(), infoContoBeneficiario.Name(), `beneficiario_natura == "PP"`)
	require.NoError(t, err)

	err = cfgOrc.AddPath(sa.Name(), infoCartaBeneficiario.Name(), `beneficiario_natura == "DT"`)
	require.NoError(t, err)

	err = cfgOrc.AddPath(infoContoBeneficiario.Name(), ea2.Name(), "")
	require.NoError(t, err)

	err = cfgOrc.AddPath(infoCartaBeneficiario.Name(), verificaIntestazioneBeneficiario.Name(), "")
	require.NoError(t, err)

	err = cfgOrc.AddPath(verificaIntestazioneBeneficiario.Name(), ea2.Name(), "")
	require.NoError(t, err)

}

var infoCartaBeneficiarioEndpointDefinition config.EndpointDefinition
var verificaIntestazioneBeneficiarioEndpointDefinition config.EndpointDefinition

func SetUpInfoCartaEndpointDefinition(t *testing.T) {

	/*
	 * infoCartaBeneficiarioEndpointDefinition
	 */
	infoCartaBeneficiarioEndpointDefinition = config.EndpointDefinition{
		Method:   http.MethodGet,
		HostName: "tpm-router-card-inquiry-api-common-card.app.coll2.ocprm.testposte",
		Scheme:   "https",
		Path:     "/listaCarte/api/v1/carta/{v:beneficiario_numero}",
		Headers: []config.NameValuePair{
			{Name: "requestId", Value: "{h:requestId}"},
			{Name: "correlationId", Value: "{h:trackId}"},
		},
		OnResponseActions: []config.OnResponseAction{
			{
				StatusCode: 200,
				ProcessVars: []config.ProcessVar{
					{
						Name:  "beneficiario_intestazioneSystem",
						Value: "{$.informazioni.titolare.intestazione}",
					},
					{
						Name:  "beneficiario_codiceFiscaleSystem",
						Value: "{$.informazioni.titolare.codiceFiscale}",
					},
					{
						Name:  "beneficiario_partitaIva",
						Value: "{$.informazioni.titolare.partitaIva}",
					},
					{
						Name:  "beneficiario_tipoClientela",
						Value: "{$.tipologia.tipoClientela}",
					},
					{
						Name:  "beneficiario_contoAssociato.servizio",
						Value: "{$.datiFinanziari.contoAssociato.servizio}",
					},
					{
						Name:  "beneficiario_contoAssociato.filiale",
						Value: "{$.datiFinanziari.contoAssociato.filiale}",
					},
					{
						Name:  "beneficiario_contoAssociato.numero",
						Value: "{$.datiFinanziari.contoAssociato.numero}",
					},
					{
						Name:  "beneficiario_contoAssociato.categoria",
						Value: "{$.datiFinanziari.contoAssociato.categoria}",
					},
				},
			},
			{
				StatusCode: -1,
				Errors: []config.ErrorInfo{
					{
						StatusCode: 0,
						Ambit:      "info-carta",
						Message:    "{$.message}",
					},
				},
			},
		},
	}

	/*
	 * verificaIntestazioneBeneficiarioEndpointDefinition
	 */
	verificaIntestazioneBeneficiarioEndpointDefinition = config.EndpointDefinition{
		Method:   http.MethodPost,
		HostName: "localhost",
		Port:     "8081",
		Scheme:   "http",
		Path:     "/api/v1/verificaIntestazione",
		Headers: []config.NameValuePair{
			{Name: "requestId", Value: "{h:requestId}"},
			{Name: "trackId", Value: "{h:trackId}"},
			{Name: "Content-type", Value: constants.ContentTypeApplicationJson},
		},
		Body: config.PostData{
			Name:          "body-verifica-intestazione-beneficiario",
			Type:          "template",
			ExternalValue: strings.Join([]string{VerificaIntestazioneBeneficiarioEndpointId + "-body", "tmpl"}, "."),
		},
		OnResponseActions: []config.OnResponseAction{
			{
				StatusCode: 200,
				ProcessVars: []config.ProcessVar{
					{
						Name:  "esitoRagioneSociale",
						Value: "{$.esitoRagioneSociale}",
					},
					{
						Name:  "indiceRagioneSociale",
						Value: "{$.indiceRagioneSociale}",
					},
				},
				Errors: []config.ErrorInfo{
					{
						Guard:      `esitoRagioneSociale != "SI" || indiceRagioneSociale <= 11`,
						StatusCode: 400,
						Ambit:      VerificaIntestazioneBeneficiarioEndpointName,
						Code:       "{$.resultCode}",
						Message:    "{$.resultMessage}",
					},
				},
			},
			{
				StatusCode: -1,
				Errors: []config.ErrorInfo{
					{
						StatusCode: 0,
						Ambit:      VerificaIntestazioneBeneficiarioEndpointName,
						Code:       "{$.resultCode}",
						Message:    "{$.resultMessage}",
					},
				},
			},
		},
	}
}

func TestConfig(t *testing.T) {

	SetUpOrchestration(t)
	SetUpInfoCartaEndpointDefinition(t)

	var deserOrc config.Orchestration

	t.Log("JSON SerDe --------------------------")
	b, err := cfgOrc.ToJSON()
	require.NoError(t, err)
	t.Log(string(b))

	// Deserialization
	deserOrc, err = config.NewOrchestrationFromJSON(b)
	require.NoError(t, err)

	b, err = deserOrc.ToJSON()
	require.NoError(t, err)
	t.Log(string(b))

	t.Log("YAML SerDe --------------------------")
	b, err = cfgOrc.ToYAML()
	require.NoError(t, err)

	err = ioutil.WriteFile(OrchestrationYAMLFileName, b, fs.ModePerm)
	require.NoError(t, err)

	// Should remove... at the moment is good this way....
	// defer os.Remove(OrchestrationYAMLFileName)

	deserOrc, err = config.NewOrchestrationFromYAML(b)
	require.NoError(t, err)

	b, err = deserOrc.ToYAML()
	require.NoError(t, err)
	t.Log(string(b))

	t.Log("InfoCarta1 --------------------------")
	b, err = yaml.Marshal(infoCartaBeneficiarioEndpointDefinition)
	require.NoError(t, err)

	err = ioutil.WriteFile(strings.Join([]string{InfoCartaBeneficiarioDTEndpointId, "yml"}, "."), b, fs.ModePerm)
	require.NoError(t, err)

	t.Log("Verifica intestazione beneficiario --------------------------")
	b, err = yaml.Marshal(verificaIntestazioneBeneficiarioEndpointDefinition)
	require.NoError(t, err)

	err = ioutil.WriteFile(strings.Join([]string{VerificaIntestazioneBeneficiarioEndpointId, "yml"}, "."), b, fs.ModePerm)
	require.NoError(t, err)

}

var serde = []byte(`
paths:
  - source: start-name
    target: echo-name
  - source: echo-name
    target: end-name
activities: 
  - activity:
      name: start-name
      type: start-activity
    property: a-start-property
  - activity:
      name: echo-name
      type: echo-activity
    message: a-message
  - activity:
      name: end-name
      type: end-activity
`)

func TestConfigSerde(t *testing.T) {
	deserOrch, err := config.NewOrchestrationFromYAML(serde)
	require.NoError(t, err)

	b, err := deserOrch.ToYAML()
	require.NoError(t, err)
	t.Log(string(b))
}

var dict = []byte(`
APBP: "ABP"
NPDB: "BPI"
APDB: "BPI"
APPP: "APP"
BPOL: "BPO"
PPAY: "PPI"
RPOL: "RPO"
`)

func TestNewDictionary(t *testing.T) {
	d, err := config.NewDictionary("causali", dict)
	require.NoError(t, err)

	t.Log(d)
}
