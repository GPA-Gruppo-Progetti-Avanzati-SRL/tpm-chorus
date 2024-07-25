package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
)

const (
	MongoActivityFindOneName                  = "mongo-activity-find-one"
	MongoActivityFindOneDescription           = MongoActivityFindOneName + descriptionSuffix
	MongoActivityFindOneRefDefinition         = "mongo-activity-find-one.yml"
	MongoActivityFindOneRefDefinitionFileName = OrchestrationWorkPath + MongoActivityFindOneRefDefinition
)

func SetUpMongoActivityFindOneMovie() config.MongoActivityDefinition {

	definition := config.MongoActivityDefinition{

		LksName:      "default",
		CollectionId: "movies",
		StatementData: map[jsonops.MongoJsonOperationStatementPart]string{
			jsonops.MongoActivityFindOneQueryProperty: `{ "year": {v:year}, "title": "{v:title}" }`,
			jsonops.MongoActivityFindOneSortProperty:  `{ "title": 1 }`,
			// jsonops.MongoActivityFindOneProjectionProperty: `{ "year": 1 }`,
			// jsonops.MongoActivityFindOneOptsProperty:       "",
		},
		OnResponseActions: []config.OnResponseAction{
			{
				StatusCode: 200,
				ProcessVars: []config.ProcessVar{
					{
						Name:  "movieFound",
						Value: ":true",
					},
					{
						Name:  "movieContextName",
						Value: MongoActivityFindOneName,
					},
				},
			},
			{
				StatusCode: 404,
				ProcessVars: []config.ProcessVar{
					{
						Name:  "movieFound",
						Value: ":false",
					},
				},
			},
			{
				StatusCode: -1,
				Errors: []config.ErrorInfo{
					{
						StatusCode: 0,
						Ambit:      MongoActivityFindOneName,
						Message:    "{v:dict,sample,500}",
					},
				},
			},
		},
	}

	return definition
}
