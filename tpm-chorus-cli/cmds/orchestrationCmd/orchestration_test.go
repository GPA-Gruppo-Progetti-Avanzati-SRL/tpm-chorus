package orchestrationCmd_test

import (
	_ "embed"
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/tpm-chorus-cli/cmds/orchestrationCmd"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"net/http"
	"strconv"
	"testing"
)

const (
	acceptHeader                     = "*/*"
	agentHeader                      = "insomnium/0.2.3"
	applicationJsonContentTypeHeader = "application/json"
)

//go:embed sample.json
var sampleBody []byte

var sampleRequestDefinitionJson = `
{
  "method": "POST",
  "path": "/test/test01/api/v1/ep01/test",
  "headers": {
    "Accept": [
      "*/*"
    ],
    "Content-Length": [
      "114"
    ],
    "Content-Type": [
      "application/json"
    ],
    "Host": [
      "localhost:8080"
    ],
    "User-Agent": [
      "insomnium/0.2.3"
    ],
    "canale": [
      "APBP"
    ],
    "requestId": [
      "bf53415b-54ce-4b5a-a470-b01943a68f89"
    ],
    "trackId": [
      "cbd5c903-b1ee-4c6e-ba39-bb040c0116f8"
    ]
  },
  "params": [
    {
      "name": "pathId",
      "value": "test"
    }
  ]
}
`
var sampleRequestDefinition = orchestrationCmd.RequestDefinition{
	Method: http.MethodPost,
	Path:   "/test/test01/api/v1/ep01/test",
	Headers: http.Header{
		"Host":           []string{"localhost:8080"},
		"Content-Type":   []string{applicationJsonContentTypeHeader},
		"User-Agent":     []string{agentHeader},
		"canale":         []string{"APBP"},
		"requestId":      []string{"bf53415b-54ce-4b5a-a470-b01943a68f89"},
		"trackId":        []string{"cbd5c903-b1ee-4c6e-ba39-bb040c0116f8"},
		"Accept":         []string{acceptHeader},
		"Content-Length": []string{strconv.Itoa(len(sampleBody))},
	},
	Params: []har.Param{{Name: "pathId", Value: "test"}},
}

func TestSampleRequetDefinition(t *testing.T) {
	b, err := json.Marshal(sampleRequestDefinition)
	require.NoError(t, err)
	t.Log(string(b))

	b, err = yaml.Marshal(sampleRequestDefinition)
	require.NoError(t, err)
	t.Log(string(b))
}
