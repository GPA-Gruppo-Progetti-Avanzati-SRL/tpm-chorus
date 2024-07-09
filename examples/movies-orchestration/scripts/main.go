package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/fs"
	"io/ioutil"
)

const (
	OrchestrationWorkPath     = "~/examples/movies-orchestration/"
	OrchestrationYAMLFileName = OrchestrationWorkPath + "tpm-orchestration.yml"

	descriptionSuffix           = " description"
	RequestActivityName         = "start-activity"
	RequestActivityDescription  = RequestActivityName + descriptionSuffix
	EchoActivityName            = "echo-activity"
	EchoActivityDescription     = EchoActivityName + descriptionSuffix
	ResponseActivityName        = "end-activity"
	ResponseActivityDescription = ResponseActivityName + descriptionSuffix

	SemLogContext = "%s --------------------------"
)

func main() {
	err := writeOrchestrationToFile(OrchestrationYAMLFileName, SetUpOrchestration())
	requireNoError(err)
}

func writeOrchestrationToFile(fn string, orc *config.Orchestration) error {

	log.Info().Msgf(SemLogContext, "Orchestration")

	b, err := yaml.Marshal(orc)
	if err != nil {
		return err
	}

	return writeToFile(fn, b)
}

func writeEndpointDefinitionToFile(activityName string, fn string, ep config.EndpointDefinition) error {

	log.Info().Msgf(SemLogContext, activityName)

	b, err := yaml.Marshal(ep)
	if err != nil {
		return err
	}

	return writeToFile(fn, b)
}

func writeProducerDefinitionToFile(activityName string, fn string, ep config.ProducerDefinition) error {

	log.Info().Msgf(SemLogContext, activityName)

	b, err := yaml.Marshal(ep)
	if err != nil {
		return err
	}

	return writeToFile(fn, b)
}

func writeToFile(fn string, b []byte) error {

	outFileName, _ := util.ResolvePath(fn)
	log.Info().Str("file-name", outFileName).Msg("producing file")
	err := ioutil.WriteFile(outFileName, b, fs.ModePerm)
	return err
}

func requireNoError(err error) {
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
