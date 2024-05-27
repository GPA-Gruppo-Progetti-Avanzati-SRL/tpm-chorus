package oapiutil

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"net/http"
	"strings"
	"tpm-chorus/smperror"
)

func ValidationError(err error) error {

	smpe := &smperror.SymphonyError{
		StatusCode: http.StatusBadRequest,
		Ambit:      "validation",
		Step:       "validation",
		ErrCode:    "OAPI-0001",
		Message:    err.Error(),
	}

	var sb strings.Builder
	if reqErr, ok := err.(*openapi3filter.RequestError); ok {

		if schErr, ok := reqErr.Err.(*openapi3.SchemaError); ok {
			sb.WriteString(schErr.Schema.Description)
			if schErr.Schema.Description != "" {
				sb.WriteString(". ")
			}
			sb.WriteString(fmt.Sprintf("(%v): ", schErr.Value))
			sb.WriteString(schErr.Reason)
		} else {
			sb.WriteString(reqErr.Reason)
		}

		if sb.String() != "" {
			smpe.Message = sb.String()
		}
	}

	return smpe
}
