package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"io/fs"
	"io/ioutil"
)

const (
	OrchestrationWorkPath                  = "~/examples/movies-orchestration/"
	OrchestrationYAMLFileName              = OrchestrationWorkPath + "tpm-orchestration.yml"
	NestedOrchestrationYAMLFileName        = OrchestrationWorkPath + "nested-orchestration/tpm-orchestration.yml"
	descriptionSuffix                      = " description"
	RequestActivityName                    = "start-activity"
	RequestActivityDescription             = RequestActivityName + descriptionSuffix
	EchoActivityName                       = "echo-activity"
	EchoActivityDescription                = EchoActivityName + descriptionSuffix
	NestedOrchestrationActivityName        = "nested-orchestration-activity"
	NestedOrchestrationActivityDescription = NestedOrchestrationActivityName + descriptionSuffix

	ResponseActivityName        = "end-activity"
	ResponseActivityDescription = ResponseActivityName + descriptionSuffix

	SemLogContext = "%s --------------------------"
)

func main() {
	err := writeOrchestrationToFile(OrchestrationYAMLFileName, SetUpOrchestration())
	requireNoError(err)

	err = writeOrchestrationToFile(NestedOrchestrationYAMLFileName, SetUpNestedOrchestration())
	requireNoError(err)

	var b []byte
	mongoDef := SetUpMongoActivityFindOneMovie()
	b, err = yaml.Marshal(mongoDef)
	requireNoError(err)
	err = writeActivityDefinitionToFile(MongoActivityFindOneName, MongoActivityFindOneRefDefinitionFileName, b)
	requireNoError(err)

	epDef := SetUpEndpoint01Definition()
	b, err = yaml.Marshal(epDef)
	requireNoError(err)
	err = writeActivityDefinitionToFile(Endpoint01ActivityName, Endpoint01EndpointFileName, b)
	requireNoError(err)

	mongoDef = SetUpMongoActivityReplaceOneMovie()
	b, err = yaml.Marshal(mongoDef)
	requireNoError(err)
	err = writeActivityDefinitionToFile(MongoActivityReplaceOneName, MongoActivityReplaceOneRefDefinitionFileName, b)
	requireNoError(err)

	mongoDef = SetUpMongoActivityAggregateOneMovie()
	b, err = yaml.Marshal(mongoDef)
	requireNoError(err)
	err = writeActivityDefinitionToFile(MongoActivityAggregateOneName, MongoActivityAggregateOneRefDefinitionFileName, b)
	requireNoError(err)

	mongoDef = SetUpMongoActivityUpdateOneMovie()
	b, err = yaml.Marshal(mongoDef)
	requireNoError(err)
	err = writeActivityDefinitionToFile(MongoActivityUpdateOneName, MongoActivityUpdateOneRefDefinitionFileName, b)
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

func writeActivityDefinitionToFile(activityName string, fn string, b []byte) error {
	log.Info().Msgf(SemLogContext, activityName)
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
