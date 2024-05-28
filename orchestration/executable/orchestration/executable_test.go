package orchestration_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

const (
	ResponseActivityName = "end-name"
	RequestActivityName  = "start-name"

	BeneficiarioDTActivityName           = "beneficiario-DT"
	BeneficiarioDTEndpoointInfoCartaId   = "infocarta-1"
	BeneficiarioDTEndpoointInfoCartaName = "info-carta"

	BeneficiarioPPActivityName = "beneficiario-PP"

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
	}
	sa.ProcessVars = procVars
	sa.Validations = []config.RequestValidation{
		{Expr: "beneficiario_numero == '8188602'"},
	}

	beneficiarioDT := config.NewEndpointActivity().WithName(BeneficiarioDTActivityName)
	beneficiarioDT.Endpoints = []config.Endpoint{
		{
			Id:         BeneficiarioDTEndpoointInfoCartaId,
			Name:       BeneficiarioDTEndpoointInfoCartaName,
			Definition: strings.Join([]string{BeneficiarioDTEndpoointInfoCartaId, "yml"}, "."),
		},
	}

	beneficiarioPP := config.NewEchoActivity().WithName(BeneficiarioPPActivityName)
	beneficiarioPP.Message = "is PP"

	ea2 := config.NewResponseActivity().WithName(ResponseActivityName)
	ea2.Responses = []config.Response{{RefSimpleResponse: "responseSimple.tmpl"}}

	cfgOrc = config.Orchestration{
		Id: "smp-o-id",
		Activities: []config.Configurable{
			sa, beneficiarioDT, beneficiarioPP, ea2,
		},
		References: config.DataReferences{
			{Path: "responseSimple.tmpl", Data: []byte(`{"msg":"hello-world"}`)},
		},
	}

	err := cfgOrc.AddPath(RequestActivityName, BeneficiarioPPActivityName, `beneficiario_natura == "PP"`)
	require.NoError(t, err)

	err = cfgOrc.AddPath(RequestActivityName, BeneficiarioDTActivityName, `beneficiario_natura == "DT"`)
	require.NoError(t, err)

	err = cfgOrc.AddPath(BeneficiarioPPActivityName, ResponseActivityName, "")
	require.NoError(t, err)

	err = cfgOrc.AddPath(BeneficiarioDTActivityName, ResponseActivityName, "")
	require.NoError(t, err)
}

func TestNewOrchestration(t *testing.T) {

	SetUpOrchestration(t)

	orc, err := orchestration.NewOrchestration(&cfgOrc)
	require.NoError(t, err)

	if !orc.IsValid() {
		t.Fatal("orchestration is invalid")
	}
	t.Log(orc)

	hs := http.Header{}
	wfc, err := wfcase.NewWorkflowCase(orc.Cfg.Id, "1.0", "sha-number", orc.Cfg.Description, nil, nil, nil)
	require.NoError(t, err)

	req, _ := har.NewRequest(http.MethodGet, "/my/path", nil, hs, nil)
	wfc.AddEndpointRequestData("request", req, config.PersonallyIdentifiableInformation{Domain: "common", AppliesTo: "req,resp"})

	a, err := orc.Execute(wfc)
	require.NoError(t, err)

	require.NotNil(t, a, "a cannot be null....")

	t.Log("Leaf activity: ", a.Name())
	t.Log(a)
}
