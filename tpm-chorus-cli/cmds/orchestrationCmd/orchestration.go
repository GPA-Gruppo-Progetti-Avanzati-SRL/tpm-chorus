package orchestrationCmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/linkedservices"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/responseactivity"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/tpm-chorus-cli/cmds"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var (
	orchestrationFolder   string
	requestDefinitionFile string
	envConfigFileName     string
)

type RequestDefinition struct {
	Method       string      `json:"method,omitempty" yaml:"method,omitempty" mapstructure:"method,omitempty"`
	Host         string      `json:"host,omitempty" yaml:"host,omitempty" mapstructure:"host,omitempty"`
	Path         string      `json:"path,omitempty" yaml:"path,omitempty" mapstructure:"path,omitempty"`
	BodyFilename string      `json:"body,omitempty" yaml:"body,omitempty" mapstructure:"body,omitempty"`
	Headers      http.Header `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers,omitempty"`
	Params       []har.Param `json:"params,omitempty" yaml:"params,omitempty" mapstructure:"params,omitempty"`
}

func newRequestDefinitionFromFilename(def string) (*har.Request, error) {
	const semLogContext = semLogContextCmd + "new-request-definition"

	b, err := os.ReadFile(def)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	r := RequestDefinition{}
	err = yaml.Unmarshal(b, &r)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	if r.BodyFilename == "" {
		err = errors.New("no file specified for body")
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	bodyFile := filepath.Join(filepath.Dir(def), r.BodyFilename)
	b, err = os.ReadFile(bodyFile)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	r.Headers["Content-Length"] = []string{strconv.Itoa(len(b))}

	req, err := har.NewRequest(
		r.Method,
		r.Path,
		b,
		r.Headers,
		r.Params,
	)

	return req, err
}

const (
	semLogContextCmd = "orchestration-cmd::"
)

// theCmd
// Sample params: -i orchestration/transform/case-001-input.json -r orchestration/transform/case-001-rule.yml -o kazaam.out.json
var theCmd = &cobra.Command{
	Use:   "orchestration",
	Short: "test of orchestrations",
	Long:  `The command allows the application to execute an orchestration for testing purposes`,
	Run: func(cmd *cobra.Command, args []string) {

		const semLogContext = semLogContextCmd + "run"

		err := loadEnvConfig(envConfigFileName)
		if err != nil {
			return
		}
		defer linkedservices.Close()

		err = validateArgs()
		if err != nil {
			return
		}

		bundle, err := repo.NewOrchestrationBundleFromFolder(orchestrationFolder)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		orchestrationDefinition, err := config.NewOrchestrationDefinitionFromBundle(&bundle)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		exec, err := orchestration.NewOrchestration(&orchestrationDefinition)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		if !exec.IsValid() {
			err = errors.New("the configured orchestration is invalid")
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		reqSpan := opentracing.SpanFromContext(context.Background())
		log.Info().Interface("span", reqSpan).Msg(semLogContext)

		wfCase, err := wfcase.NewWorkflowCase(exec.Cfg.Id, exec.Cfg.Version, exec.Cfg.SHA, exec.Cfg.Description, exec.Cfg.Dictionaries, exec.Cfg.References, reqSpan)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		req, err := newRequestDefinitionFromFilename(requestDefinitionFile)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		err = wfCase.AddEndpointRequestData(config.InitialRequestContextNameStringReference, req, exec.Cfg.PII)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

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
					config.InitialRequestContextNameStringReference,
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
			err = wfCase.AddEndpointResponseData(
				config.InitialRequestContextNameStringReference,
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
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		har := wfCase.GetHarData(wfcase.ReportLogHAR, nil)
		b, err := json.Marshal(har)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		log.Info().Msg(semLogContext + " ending")

		fmt.Println("---------- BOF HAR orchetration log")
		fmt.Println(string(b))
		fmt.Println("---------- EOF HAR orchetration log")
	},
}

func validateArgs() error {
	const semLogContext = semLogContextCmd + "validate-args"
	var err error

	if orchestrationFolder == "" || !util.FileExists(orchestrationFolder) {
		err = errors.New("error: orchestration folder must be provided and should exists")
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	if requestDefinitionFile == "" || !util.FileExists(requestDefinitionFile) {
		err = errors.New("error: a file containing the request definition should be provided and should exists")
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	if envConfigFileName == "" || !util.FileExists(envConfigFileName) {
		err = errors.New("error: a file containing environment definitions (linked services and others should be provided and should exists")
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
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

func init() {
	cmds.RootCmd.AddCommand(theCmd)
	theCmd.Flags().StringVarP(&orchestrationFolder, "orc", "o", "", "folder containing orchestration configs")
	theCmd.Flags().StringVarP(&requestDefinitionFile, "req", "r", "", "a file containing the definition of the request")
	theCmd.Flags().StringVarP(&envConfigFileName, "cfg", "c", "", "a file containing environment configs")
}

type AppConfig struct {
	Services *linkedservices.Config                `yaml:"linked-services" mapstructure:"linked-services" json:"linked-services"`
	Metrics  map[string]promutil.MetricGroupConfig `yaml:"metrics,omitempty" mapstructure:"metrics,omitempty" json:"metrics,omitempty"`
}

func loadEnvConfig(fn string) error {

	const semLogContext = semLogContextCmd + "load-env-config"

	b, err := os.ReadFile(fn)
	if err != nil {
		log.Err(err).Err(err).Msg(semLogContext)
		return err
	}

	cfg := AppConfig{}
	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	err = transform.InitializeKazaamRegistry()
	if nil != err {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	err = linkedservices.InitRegistry(cfg.Services)
	if nil != err {
		log.Error().Err(err).Msg(semLogContext + " linked services initialization error")
		return err
	}
	defer linkedservices.Close()

	_, err = promutil.InitRegistry(cfg.Metrics)
	if nil != err {
		log.Error().Err(err).Msg(semLogContext + " metrics registry initialization error")
		return err
	}

	return nil
}
