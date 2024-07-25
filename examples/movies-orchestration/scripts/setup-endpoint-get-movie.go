package main

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"net/http"
	"time"
)

const (
	Endpoint01ActivityName        = "endpoint-get-movie"
	Endpoint01ActivityDescription = Endpoint01ActivityName + descriptionSuffix
	Endpoint01EndpointId          = "endpoint-get-movie-1"
	Endpoint01EndpointName        = "endpoint-get-movie"
	Endpoint01EndpointDescription = Endpoint01EndpointId + descriptionSuffix
	Endpoint01EndpointFileName    = OrchestrationWorkPath + Endpoint01EndpointId + ".yml"
)

func SetUpEndpoint01Definition() config.EndpointDefinition {

	endpointDefinition := config.EndpointDefinition{
		Method:   http.MethodGet,
		HostName: "localhost",
		Port:     "3007",
		Scheme:   "http",
		Path:     "/movies/{$.year,sprf=.0f}/{$.title}",
		HttpClientOptions: &config.HttpClientOptions{
			RestTimeout: 10 * time.Second,
		},
		Headers: []config.NameValuePair{
			{Name: "requestId", Value: "{h:requestId}"},
			{Name: "correlationId", Value: "{h:trackId}"},
		},
		OnResponseActions: []config.OnResponseAction{
			{
				StatusCode: 200,
				ProcessVars: []config.ProcessVar{
					{
						Name:  "dvd_code",
						Value: "{$.tomatoes.dvd}",
					},
					{
						Name:  "movieContextName",
						Value: Endpoint01ActivityName,
					},
				},
			},
			{
				StatusCode: 503,
				Errors: []config.ErrorInfo{
					{
						StatusCode: http.StatusInternalServerError,
						Ambit:      "service-unavailable",
						Message:    "internal server error {v:title}",
					},
				},
			},
			{
				StatusCode: -1,
				Errors: []config.ErrorInfo{
					{
						StatusCode: 0,
						Ambit:      Endpoint01ActivityName,
						Message:    "{v:dict,sample,500}",
					},
				},
			},
		},
	}

	return endpointDefinition
}
