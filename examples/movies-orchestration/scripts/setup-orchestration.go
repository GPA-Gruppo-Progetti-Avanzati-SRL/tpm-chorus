package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
	"strings"
)

func SetUpOrchestration() *config.Orchestration {

	startActivity := config.NewRequestActivity().WithName(RequestActivityName).WithDescription(RequestActivityDescription)
	procVars := []config.ProcessVar{
		{
			Name:  "year",
			Value: "{$.year,sprf=.0f}",
			Type:  "number",
		},
		{
			Name:  "title",
			Value: "{$.title}",
			Type:  "string",
		},
	}
	startActivity.ProcessVars = procVars

	mongoFindOneActivity := config.NewMongoActivity().
		WithName(MongoActivityFindOneName).
		WithDescription(MongoActivityFindOneDescription).
		WithRefDefinition(MongoActivityFindOneRefDefinition).
		WithOpType(jsonops.FindOneOperationType)

	getMovieEndpoint := config.NewEndpointActivity().WithName(Endpoint01ActivityName).WithDescription(Endpoint01ActivityDescription)
	getMovieEndpoint.En = "true"
	getMovieEndpoint.Endpoints = []config.Endpoint{
		{
			Id:          Endpoint01EndpointId,
			Name:        Endpoint01EndpointName,
			Description: Endpoint01EndpointDescription,
			Definition:  strings.Join([]string{Endpoint01EndpointId, "yml"}, "."),
		},
	}

	mongoReplaceOneActivity := config.NewMongoActivity().
		WithName(MongoActivityReplaceOneName).
		WithDescription(MongoActivityReplaceOneDescription).
		WithRefDefinition(MongoActivityReplaceOneRefDefinition).
		WithOpType(jsonops.ReplaceOneOperationType)

	mongoAggregateOneActivity := config.NewMongoActivity().
		WithName(MongoActivityAggregateOneName).
		WithDescription(MongoActivityAggregateOneDescription).
		WithRefDefinition(MongoActivityAggregateOneRefDefinition).
		WithOpType(jsonops.AggregateOneOperationType)

	nestedOrchestrationActivity := config.NewNestedOrchestrationActivity().
		WithName(NestedOrchestrationActivityName).
		WithDescription(NestedOrchestrationActivityDescription)

	endActivity := config.NewResponseActivity().
		WithName(ResponseActivityName).
		WithDescription(ResponseActivityDescription).
		WithExpressionContext(":movieContextName")

	r := config.Response{
		Id:         "app1",
		Guard:      `year == 1939`,
		StatusCode: 200,
		Headers: []config.NameValuePair{
			{
				Name:  "smp-id",
				Value: "{v:" + wfcase.SymphonyOrchestrationIdProcessVar + "}",
			},
			{
				Name:  "smp-descr",
				Value: "{v:" + wfcase.SymphonyOrchestrationDescriptionProcessVar + "}",
			},
		},
		RefSimpleResponse: strings.Join([]string{ResponseActivityName + "-body", "tmpl"}, "."),
	}
	endActivity.Responses = append(endActivity.Responses, r)

	r = config.Response{
		Id:         "otherwise",
		StatusCode: 200,
		Headers: []config.NameValuePair{
			{
				Name:  "smp-id",
				Value: "{v:" + wfcase.SymphonyOrchestrationIdProcessVar + "}",
			},
			{
				Name:  "smp-descr",
				Value: "{v:" + wfcase.SymphonyOrchestrationDescriptionProcessVar + "}",
			},
		},
		RefSimpleResponse: strings.Join([]string{ResponseActivityName + "-body", "tmpl"}, "."),
	}
	endActivity.Responses = append(endActivity.Responses, r)

	orc := &config.Orchestration{
		Id:          "movies-orchestration",
		Description: "movies-orchestration description",
		Activities: []config.Configurable{
			startActivity,
			mongoFindOneActivity,
			getMovieEndpoint,
			mongoReplaceOneActivity,
			mongoAggregateOneActivity,
			nestedOrchestrationActivity,
			endActivity,
		},
		References: config.DataReferences{
			{Path: "responseSimple.tmpl", Data: []byte(`{"msg":"hello-world"}`)},
		},
	}

	var err error
	err = orc.AddPath(startActivity.Name(), mongoFindOneActivity.Name(), "")
	requireNoError(err)

	// On not found invoke endpoint
	err = orc.AddPath(mongoFindOneActivity.Name(), getMovieEndpoint.Name(), "!movieFound")
	requireNoError(err)

	// replace mongo data
	err = orc.AddPath(getMovieEndpoint.Name(), mongoReplaceOneActivity.Name(), "")
	requireNoError(err)

	// back to default path
	err = orc.AddPath(mongoReplaceOneActivity.Name(), mongoAggregateOneActivity.Name(), "")
	requireNoError(err)

	// default path if already on db
	err = orc.AddPath(mongoFindOneActivity.Name(), mongoAggregateOneActivity.Name(), "movieFound")
	requireNoError(err)

	err = orc.AddPath(mongoAggregateOneActivity.Name(), nestedOrchestrationActivity.Name(), "")
	requireNoError(err)

	err = orc.AddPath(nestedOrchestrationActivity.Name(), endActivity.Name(), "")
	requireNoError(err)

	return orc
}

func SetUpNestedOrchestration() *config.Orchestration {

	startActivity := config.NewRequestActivity().WithName(RequestActivityName).WithDescription(RequestActivityDescription)
	procVars := []config.ProcessVar{
		{
			Name:  "year",
			Value: "{$.year}",
			Type:  "number",
		},
	}
	startActivity.ProcessVars = procVars

	echoActivity := config.NewEchoActivity().WithName(EchoActivityName).WithDescription(EchoActivityDescription)

	endActivity := config.NewResponseActivity().WithName(ResponseActivityName).WithDescription(ResponseActivityDescription)
	r := config.Response{
		Id:         "app1",
		Guard:      `year == 1939`,
		StatusCode: 200,
		Headers: []config.NameValuePair{
			{
				Name:  "smp-id",
				Value: "{v:" + wfcase.SymphonyOrchestrationIdProcessVar + "}",
			},
			{
				Name:  "smp-descr",
				Value: "{v:" + wfcase.SymphonyOrchestrationDescriptionProcessVar + "}",
			},
		},
		RefSimpleResponse: strings.Join([]string{ResponseActivityName + "-body", "tmpl"}, "."),
	}
	endActivity.Responses = append(endActivity.Responses, r)

	r = config.Response{
		Id:         "otherwise",
		StatusCode: 200,
		Headers: []config.NameValuePair{
			{
				Name:  "smp-id",
				Value: "{v:" + wfcase.SymphonyOrchestrationIdProcessVar + "}",
			},
			{
				Name:  "smp-descr",
				Value: "{v:" + wfcase.SymphonyOrchestrationDescriptionProcessVar + "}",
			},
		},
		RefSimpleResponse: strings.Join([]string{ResponseActivityName + "-body", "tmpl"}, "."),
	}
	endActivity.Responses = append(endActivity.Responses, r)

	orc := &config.Orchestration{
		Id:          "example-01-orc-01",
		Description: "example-01-orc-01 description",
		Activities: []config.Configurable{
			startActivity,
			echoActivity,
			endActivity,
		},
		References: config.DataReferences{
			{Path: "responseSimple.tmpl", Data: []byte(`{"msg":"hello-world"}`)},
		},
	}

	err := orc.AddPath(startActivity.Name(), echoActivity.Name(), "")
	requireNoError(err)

	err = orc.AddPath(echoActivity.Name(), endActivity.Name(), "")
	requireNoError(err)

	return orc
}
