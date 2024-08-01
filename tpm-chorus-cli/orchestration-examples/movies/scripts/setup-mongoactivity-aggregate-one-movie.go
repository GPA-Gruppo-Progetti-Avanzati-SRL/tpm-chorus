package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
)

const (
	MongoActivityAggregateOneName                  = "mongo-activity-aggregate-one"
	MongoActivityAggregateOneDescription           = MongoActivityAggregateOneName + descriptionSuffix
	MongoActivityAggregateOneRefDefinition         = "mongo-activity-aggregate-one.yml"
	MongoActivityAggregateOneRefDefinitionFileName = OrchestrationWorkPath + MongoActivityAggregateOneRefDefinition
)

func SetUpMongoActivityAggregateOneMovie() config.MongoActivityDefinition {

	definition := config.MongoActivityDefinition{

		LksName:      "default",
		CollectionId: "movies",
		StatementData: map[jsonops.MongoJsonOperationStatementPart]string{
			jsonops.MongoActivityAggregateOnePipelineProperty: "aggregate-one-pipeline.tmpl",
			// jsonops.MongoActivityFindOneOptsProperty:       "",
		},
		OnResponseActions: []config.OnResponseAction{
			{
				StatusCode: 200,
			},
			{
				StatusCode: -1,
				Errors: []config.ErrorInfo{
					{
						StatusCode: 0,
						Ambit:      MongoActivityAggregateOneName,
						Message:    "{v:dict,sample,500}",
					},
				},
			},
		},
	}

	return definition
}
