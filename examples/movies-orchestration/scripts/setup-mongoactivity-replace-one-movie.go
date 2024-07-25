package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
)

const (
	MongoActivityReplaceOneName                  = "mongo-activity-replace-one"
	MongoActivityReplaceOneDescription           = MongoActivityReplaceOneName + descriptionSuffix
	MongoActivityReplaceOneRefDefinition         = "mongo-activity-replace-one.yml"
	MongoActivityReplaceOneRefDefinitionFileName = OrchestrationWorkPath + MongoActivityReplaceOneRefDefinition
)

func SetUpMongoActivityReplaceOneMovie() config.MongoActivityDefinition {

	definition := config.MongoActivityDefinition{

		LksName:      "default",
		CollectionId: "movies",
		StatementData: map[jsonops.MongoJsonOperationStatementPart]string{
			jsonops.MongoActivityReplaceOneFilterProperty:      `{ "year": {v:year}, "title": "{v:title}" }`,
			jsonops.MongoActivityReplaceOneReplacementProperty: `{$.}`,
			jsonops.MongoActivityReplaceOneOptsProperty:        `{ "upsert": true }`,
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
						Ambit:      MongoActivityReplaceOneName,
						Message:    "{v:dict,sample,500}",
					},
				},
			},
		},
	}

	return definition
}
