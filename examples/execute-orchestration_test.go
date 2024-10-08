package examples_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/responseactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

//go:embed movies-sample.json
var movieSample []byte

const orchestration1Folder = "./movies-orchestration"

func TestExecuteMoviesOrchestration(t *testing.T) {

	const semLogContext = "examples::test-execute-orchestration-1"

	bundle, err := repo.NewOrchestrationBundleFromFolder(orchestration1Folder)
	require.NoError(t, err)

	orchestrationDefinition, err := config.NewOrchestrationDefinitionFromBundle(&bundle)
	require.NoError(t, err)

	exec, err := orchestration.NewOrchestration(&orchestrationDefinition)
	require.NoError(t, err)
	require.True(t, exec.IsValid(), "the configured orchestration is invalid")

	reqSpan := opentracing.SpanFromContext(context.Background())
	log.Info().Interface("span", reqSpan).Msg(semLogContext)

	wfCase, err := wfcase.NewWorkflowCase(exec.Cfg.Id, exec.Cfg.Version, exec.Cfg.SHA, exec.Cfg.Description, exec.Cfg.Dictionaries, exec.Cfg.References, nil, reqSpan)
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
		movieSample,
		http.Header{
			"Host":           []string{"localhost:8080"},
			"Content-Type":   []string{constants.ContentTypeApplicationJson},
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

	err = wfCase.SetHarEntryRequest(wfcase.InitialRequestHarEntryId, req, exec.Cfg.PII)
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
			_ = wfCase.SetHarEntryResponse(
				wfcase.InitialRequestHarEntryId,
				har.NewResponse(
					resp.Status, resp.StatusText,
					resp.Content.MimeType, resp.Content.Data,
					resp.Headers,
				),
				exec.Cfg.PII)
			log.Info().Str("response", string(resp.Content.Data)).Msg(semLogContext)
		}
	}

	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		sc, ct, resp := produceErrorResponse(err)
		err = wfCase.SetHarEntryResponse(
			wfcase.InitialRequestHarEntryId,
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

	har := wfCase.GetHarData(wfcase.ReportLogHAR, nil)
	b, err := json.Marshal(har)
	require.NoError(t, err)
	t.Log(string(b))
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
