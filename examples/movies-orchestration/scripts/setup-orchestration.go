package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"strings"
)

func SetUpOrchestration() *config.Orchestration {

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
