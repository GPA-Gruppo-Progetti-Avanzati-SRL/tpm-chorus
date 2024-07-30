package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
)

const (
	MongoActivityUpdateOneName                  = "mongo-activity-update-one"
	MongoActivityUpdateOneDescription           = MongoActivityUpdateOneName + descriptionSuffix
	MongoActivityUpdateOneRefDefinition         = "mongo-activity-update-one.yml"
	MongoActivityUpdateOneRefDefinitionFileName = OrchestrationWorkPath + MongoActivityUpdateOneRefDefinition
)

func SetUpMongoActivityUpdateOneMovie() config.MongoActivityDefinition {

	definition := config.MongoActivityDefinition{

		LksName:      "default",
		CollectionId: "movies",
		StatementData: map[jsonops.MongoJsonOperationStatementPart]string{
			jsonops.MongoActivityUpdateOneFilterProperty: `{ "hello": "world" }`,
			jsonops.MongoActivityUpdateOneUpdateProperty: "update-one-update-document.tmpl",
			jsonops.MongoActivityUpdateOneOptsProperty:   `{ "upsert": true }`,
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
						Ambit:      MongoActivityUpdateOneName,
						Message:    "{v:dict,sample,500}",
					},
				},
			},
		},
	}

	return definition
}
