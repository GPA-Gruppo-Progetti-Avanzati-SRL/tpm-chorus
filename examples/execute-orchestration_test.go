package examples_test

import (
	"context"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/responseactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

const orchestration1MountPoint = "./open-api-repo-example-01/orchestration1"

func TestExecuteOrchestration1(t *testing.T) {

	const semLogContext = "examples::test-execute-orchestration-1"

	cfg := &registry.Config{
		MountPoint: orchestration1MountPoint,
	}

	reg, err := registry.LoadRegistry(cfg)
	require.NoError(t, err)

	e := reg.Entries[0]
	e.OrchestrationBundle.ShowInfo()

	exec, err := orchestration.NewOrchestration(&e.Orchestrations[0])
	require.NoError(t, err)
	require.True(t, exec.IsValid(), "the configured orchestration is invalid")

	reqSpan := opentracing.SpanFromContext(context.Background())
	log.Info().Interface("span", reqSpan).Msg(semLogContext)

	wfCase, err := wfcase.NewWorkflowCase(exec.Cfg.Id, exec.Cfg.Version, exec.Cfg.SHA, exec.Cfg.Description, exec.Cfg.Dictionaries, exec.Cfg.References, reqSpan)
	require.NoError(t, err)

	//  POST /test/test01/api/v1/ep01/test HTTP/1.1
	//  Host: localhost:8080
	//  Content-Type: application/json
	//  User-Agent: insomnium/0.2.3
	//  canal: APBP
	//  requestId:  bf53415b-54ce-4b5a-a470-b01943a68f89
	//  trackId: cbd5c903-b1ee-4c6e-ba39-bb040c0116f8
	//  Accept: "* / *"
	//  Content-Length: 251
	req, err := har.NewRequest(
		"POST",
		"/test/test01/api/v1/ep01/test",
		[]byte(`
{
  "canale": "APBP",
  "dataOperazione": "20240528",
  "ordinante": {
    "natura": "DR",
    "tipologia": "ALIAS",
    "numero": "123456",
    "codiceFiscale": "SSSMMM55F28E345Z",
    "intestazione": "Dario Intesta"
  }
}
`),
		http.Header{
			"Host":           []string{"localhost:8080"},
			"Content-Type":   []string{"application/json"},
			"User-Agent":     []string{"insomnium/0.2.3"},
			"canale":         []string{"APBP"},
			"requestId":      []string{"bf53415b-54ce-4b5a-a470-b01943a68f89"},
			"trackId":        []string{"cbd5c903-b1ee-4c6e-ba39-bb040c0116f8"},
			"Accept":         []string{"*/*"},
			"Content-Length": []string{"251"},
		},
		[]har.Param{{Name: "pathId", Value: "test"}},
	)
	require.NoError(t, err)

	err = wfCase.AddEndpointRequestData("request", req, exec.Cfg.PII)
	require.NoError(t, err)

	finalExec, err := exec.Execute(wfCase)

	if err == nil {
		respExec, isResponseActivity := finalExec.(*responseactivity.ResponseActivity)
		if !isResponseActivity {
			log.Fatal().Err(errors.New("final activity is not a response activity")).Msg(semLogContext)
		}

		var resp *har.Response
		resp, err = respExec.ResponseJSON(wfCase)
		if err == nil {
			_ = wfCase.AddEndpointResponseData(
				"request",
				har.NewResponse(
					resp.Status, resp.StatusText,
					resp.Content.MimeType, resp.Content.Data,
					resp.Headers,
				),
				exec.Cfg.PII)
			log.Error().Str("response", string(resp.Content.Data)).Msg(semLogContext)
		}
	}

	if err != nil {
		sc, ct, resp := produceErrorResponse(err)
		err = wfCase.AddEndpointResponseData(
			"request",
			har.NewResponse(
				sc, "execution error",
				constants.ContentTypeApplicationJson, resp,
				[]har.NameValuePair{{Name: constants.ContentTypeHeader, Value: ct}},
			),
			exec.Cfg.PII)
		log.Error().Str("response", string(resp)).Msg(semLogContext)
	}

	bndry, ok := exec.Cfg.FindBoundaryByName(config.DefaultActivityBoundary)
	if ok {
		err = exec.ExecuteBoundary(wfCase, bndry)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}
	}

	require.NoError(t, err)
}

func produceErrorResponse(err error) (int, string, []byte) {
	var exeErr *smperror.SymphonyError
	ok := errors.As(err, &exeErr)

	if !ok {
		exeErr = smperror.NewExecutableServerError(smperror.WithStep("not-applicable"), smperror.WithErrorAmbit("general"), smperror.WithErrorMessage(err.Error()))
	}

	response := []byte("{ \"err\": -99 }")
	response, err = exeErr.ToJSON(response)
	if err != nil {
		response = []byte("{ \"err\": -99 }")
	}

	return exeErr.StatusCode, constants.ContentTypeApplicationJson, response
}
