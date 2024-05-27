package oapiutil

import "github.com/getkin/kin-openapi/openapi3"

func ServersUrl(doc *openapi3.T) string {
	if len(doc.Servers) > 0 {
		u := doc.Servers[0].URL
		return u
	}

	return ""
}
