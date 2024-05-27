package oapiutil

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
)

const (
	ExamplesResponseKey = "response"
)

func RetrieveExample(doc *openapi3.T, method string, oapiPath string, statusCode int, contentType string) ([]byte, string, error) {

	p := doc.Paths.Find(oapiPath)
	if p == nil {
		return nil, "", fmt.Errorf("the path %s cannot be found", oapiPath)
	}

	op := p.GetOperation(method)
	if op == nil {
		return nil, "", fmt.Errorf("the operation %s in path %s cannot be found", method, oapiPath)
	}

	resp := op.Responses.Status(statusCode)
	if resp == nil {
		return nil, "", fmt.Errorf("the response for %d in operation %s in path %s cannot be found", statusCode, method, oapiPath)
	}

	mediaType := resp.Value.Content.Get(contentType)
	if mediaType == nil {
		return nil, "", fmt.Errorf("media type %s in the response for %d in operation %s in path %s cannot be found", contentType, statusCode, method, oapiPath)
	}

	if mediaType.Examples == nil || mediaType.Examples[ExamplesResponseKey] == nil || mediaType.Examples[ExamplesResponseKey].Value == nil {
		return nil, "", fmt.Errorf("example in media type %s in the response for %d in operation %s in path %s are not present", contentType, statusCode, method, oapiPath)
	}

	if mediaType.Examples[ExamplesResponseKey].Value.ExternalValue != "" {
		return nil, mediaType.Examples[ExamplesResponseKey].Value.ExternalValue, nil
	}

	if mediaType.Examples[ExamplesResponseKey].Value.Value != nil {
		theExample := mediaType.Examples[ExamplesResponseKey].Value.Value

		b, err := json.Marshal(theExample)
		if err != nil {
			return nil, "", err
		}

		return b, "", nil
	}

	return nil, "", fmt.Errorf("example in media type %s in the response for %d in operation %s in path %s are not present", contentType, statusCode, method, oapiPath)
}
