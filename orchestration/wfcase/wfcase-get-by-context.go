package wfcase

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

func (wfc *WfCase) GetRequestFromContext(ctxName ResolverContextReference, method, url string) (*har.Request, error) {
	const semLogContext = "wf-case::get-request-from-context"
	body, err := wfc.GetBodyByContext(ctxName, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	headers := wfc.GetHeadersFromContext(ctxName.Name)
	var httpHeaders http.Header
	for _, h := range headers {
		if httpHeaders == nil {
			httpHeaders = make(http.Header)
		}
		httpHeaders[h.Name] = []string{h.Value}
	}

	req, err := har.NewRequest(method, url, body, httpHeaders, nil)
	return req, err
}

func (wfc *WfCase) GetHeaderFromContext(ctxName string, hn string) string {
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		v := ""
		if ctxName == config.InitialRequestContextNameStringReference {
			v = endpointData.Request.Headers.GetFirst(hn).Value
		} else {
			v = endpointData.Response.Headers.GetFirst(hn).Value
		}

		return v
	}

	return ""
}

func (wfc *WfCase) GetHeadersFromContext(ctxName string) har.NameValuePairs {
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		if ctxName == config.InitialRequestContextNameStringReference {
			return endpointData.Request.Headers
		} else {
			return endpointData.Response.Headers
		}
	}

	return nil
}

func (wfc *WfCase) GetBodyByContext(resolverContext ResolverContextReference, ignoreNonApplicationJsonResponseContent bool) ([]byte, error) {
	var b []byte
	var err error
	if entry, ok := wfc.Entries[resolverContext.Name]; ok {
		if resolverContext.UseResponse {
			if entry.Response.Content != nil {
				if strings.HasPrefix(entry.Response.Content.MimeType, constants.ContentTypeApplicationJson) {
					b = entry.Response.Content.Data
				} else {
					if ignoreNonApplicationJsonResponseContent {
						return nil, nil
					}

					return nil, errors.New("content type is not application/json")
				}
			}
		} else {
			if entry.Request.PostData != nil {
				if strings.HasPrefix(entry.Request.PostData.MimeType, constants.ContentTypeApplicationJson) {
					b = entry.Request.PostData.Data
				} else {
					if ignoreNonApplicationJsonResponseContent {
						return nil, nil
					}

					return nil, errors.New("content type is not application/json")
				}
			}
		}
	} else {
		return nil, fmt.Errorf("cannot find ctxName %s in case", resolverContext.Name)
	}

	return b, err
}
