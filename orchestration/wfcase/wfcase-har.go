package wfcase

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/jsonmask"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	InitialRequestHarEntryId = "request"
)

var InitialRequestHarEntryReference = HarEntryReference{Name: InitialRequestHarEntryId}

type HarEntryReference struct {
	Name        string
	UseResponse bool
}

func (ref HarEntryReference) String() string {
	reqResp := "req"
	if ref.UseResponse {
		reqResp = "resp"
	}
	return fmt.Sprintf("%s-%s", ref.Name, reqResp)
}

func (wfc *WfCase) ResolveHarEntryReferenceByName(n string) (HarEntryReference, error) {
	const semLogContext = "wf-case::resolve-expression-context"
	if n == "" {
		return InitialRequestHarEntryReference, nil
	}

	var ok bool
	n, ok = IsExpression(n)
	if ok {
		v, err := wfc.Vars.EvalToString(n)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return HarEntryReference{Name: n, UseResponse: n != InitialRequestHarEntryId}, err
		}

		return HarEntryReference{Name: v, UseResponse: v != InitialRequestHarEntryId}, nil
	}

	return HarEntryReference{Name: n, UseResponse: n != InitialRequestHarEntryId}, nil
}

func (wfc *WfCase) GetHarEntry(entryId string) (*har.Entry, error) {
	if endpointData, ok := wfc.Entries[entryId]; ok {
		return endpointData, nil
	}
	return nil, fmt.Errorf("cannot find ctxName %s in case", entryId)
}

type ReportLogDetail string

const (
	ReportLogHAR        ReportLogDetail = "har"
	ReportLogHARRequest                 = "har-request"
	ReportLogSimple                     = "simple"
)

func (wfc *WfCase) GetHarData(detail ReportLogDetail, jm *jsonmask.JsonMask) *har.HAR {

	const semLogContext = "wf-case::get-har-data"
	podName := os.Getenv("HOSTNAME")
	if podName == "" {
		log.Warn().Msg(semLogContext + " HOSTNAME env variable not set")
		podName = "localhost"
	}

	har := har.HAR{
		Log: &har.Log{
			Version: "1.1",
			Creator: &har.Creator{
				Name:    "tpm-symphony",
				Version: util.GetVersion(),
			},
			Browser: wfc.Browser,
			Comment: wfc.Id,
		},
	}

	for n, e := range wfc.Entries {

		incl := true
		if detail == ReportLogHARRequest && e.Comment != InitialRequestHarEntryId {
			incl = false
		}

		if incl {

			err := e.MaskRequestBody(jm)
			if err != nil {
				log.Error().Err(err).Str("request-id", wfc.GetRequestId()).Msg("error masking request sensitive data")
			}

			err = e.MaskResponseBody(jm)
			if err != nil {
				log.Error().Err(err).Str("request-id", wfc.GetRequestId()).Msg("error masking response sensitive data")
			}

			log.Trace().Str("id", n).Msg("adding entry to har")
			har.Log.Entries = append(har.Log.Entries, e)
		} else {
			log.Trace().Str("id", n).Msg("skipping entry from har")
		}
	}

	return &har
}

func (wfc *WfCase) SetHarEntry(id string, entry *har.Entry) error {
	wfc.Entries[id] = entry
	return nil
}

func (wfc *WfCase) SetHarEntryRequest(id string, req *har.Request, pii config.PersonallyIdentifiableInformation) error {
	const semLogContext = "wf-case::set-har-entry-request"

	e, ok := wfc.Entries[id]
	if !ok {
		now := time.Now()
		e = &har.Entry{
			Comment:         id,
			StartedDateTime: now.Format("2006-01-02T15:04:05.999999999Z07:00"),
			StartDateTimeTm: now,
		}
	}
	e.Request = req
	e.PII = har.PersonallyIdentifiableInformation{
		Domain:    pii.Domain,
		AppliesTo: pii.AppliesTo,
	}
	wfc.Entries[id] = e
	return nil
}

func (wfc *WfCase) SetHarEntryResponse(id string, resp *har.Response, pii config.PersonallyIdentifiableInformation) error {

	const semLogContext = "wf-case::set-har-entry-response"

	// epData := EndpointData{Id: id, Body: body, Headers: headers, Params: params}
	// wfc.EpInfo[epData.Id] = epData
	e, ok := wfc.Entries[id]
	if !ok {
		now := time.Now()
		e = &har.Entry{
			StartDateTimeTm: now,
			StartedDateTime: now.Format("2006-01-02T15:04:05.999999999Z07:00"),
		}
	} else {
		if e.StartedDateTime != "" {
			elapsed := time.Since(e.StartDateTimeTm)
			e.Time = float64(elapsed.Milliseconds())
			log.Trace().Interface("st", e.StartDateTimeTm).Dur("elapsed", elapsed).Msg(semLogContext)
		}
	}

	e.Timings = &har.Timings{
		Blocked: -1,
		DNS:     -1,
		Connect: -1,
		Send:    -1,
		Wait:    e.Time,
		Receive: -1,
		Ssl:     -1,
	}

	e.Response = resp
	e.PII = har.PersonallyIdentifiableInformation{
		Domain:    pii.Domain,
		AppliesTo: pii.AppliesTo,
	}

	wfc.Entries[id] = e

	/*
		var hs []har.NameValuePair
		for n, h := range headers {
			for i := range h {
				hs = append(hs, har.NameValuePair{Name: n, Value: h[i]})
			}
		}

		bodyMimeType := constants.ContentTypeApplicationJson
		ct := headers.Get("Content-Type")
		if ct != "" {
			bodyMimeType = ct
		}

		e.Response = &har.Response{
			HTTPVersion: "1.1",
			Headers:     hs,
			HeadersSize: -1,
			Status:      int64(sc),
			BodySize:    int64(len(body)),
			Cookies:     []har.Cookie{},
			Content: &har.Content{
				MimeType: bodyMimeType,
				Size:     int64(len(body)),
				Data:     body,
			},
		}

		wfc.Entries[id] = e

	*/
	return nil
}

func (wfc *WfCase) NewHarRequestFromHarEntryReference(ctxName HarEntryReference, method, url string) (*har.Request, error) {
	const semLogContext = "wf-case::get-request-from-context"
	body, err := wfc.GetBodyInHarEntry(ctxName, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	headers := wfc.GetHeadersInHarEntry(ctxName.Name)
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

func (wfc *WfCase) GetHeaderInHarEntry(ctxName string, hn string) string {
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		v := ""
		if ctxName == InitialRequestHarEntryId {
			v = endpointData.Request.Headers.GetFirst(hn).Value
		} else {
			v = endpointData.Response.Headers.GetFirst(hn).Value
		}

		return v
	}

	return ""
}

func (wfc *WfCase) GetHeadersInHarEntry(ctxName string) har.NameValuePairs {
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		if ctxName == InitialRequestHarEntryId {
			return endpointData.Request.Headers
		} else {
			return endpointData.Response.Headers
		}
	}

	return nil
}

func (wfc *WfCase) GetBodyInHarEntry(resolverContext HarEntryReference, ignoreNonApplicationJsonResponseContent bool) ([]byte, error) {
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
