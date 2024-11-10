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
	if HarEntryIdIsIndexed(entryId) {
		return wfc.GetHarEntry4IndexedEntryId(entryId)
	}

	return wfc.GetHarEntry4NotIndexedEntryId(entryId)
}

func HarEntryIdIsIndexed(entryId string) bool {
	ndx := strings.Index(entryId, "#")
	return ndx >= 0
}

func (wfc *WfCase) GetHarEntry4NotIndexedEntryId(entryId string) (*har.Entry, error) {
	instanceId, err := wfc.ComputeLastUsedIndexedHarEntryId(entryId)
	if err != nil {
		return nil, err
	}

	return wfc.Entries[instanceId], nil
}

func (wfc *WfCase) GetHarEntry4IndexedEntryId(entryId string) (*har.Entry, error) {
	if endpointData, ok := wfc.Entries[entryId]; ok {
		return endpointData, nil
	}
	return nil, fmt.Errorf("cannot find ctxName %s in case", entryId)
}

func (wfc *WfCase) ComputeFirstAvailableIndexedHarEntryId(entryId string) string {

	// Search first available id.
	if HarEntryIdIsIndexed(entryId) {
		entryId = entryId[:strings.Index(entryId, "#")]
	}

	instanceNumber := 0
	instanceId := fmt.Sprintf("%s#%d", entryId, instanceNumber)
	_, ok := wfc.Entries[instanceId]
	for ok {
		instanceNumber += 1
		instanceId = fmt.Sprintf("%s#%d", entryId, instanceNumber)
		_, ok = wfc.Entries[instanceId]
	}

	return instanceId
}

func (wfc *WfCase) ComputeLastUsedIndexedHarEntryId(entryId string) (string, error) {
	var lastInstanceId string

	if HarEntryIdIsIndexed(entryId) {
		entryId = entryId[:strings.Index(entryId, "#")]
	}

	instanceNumber := 0
	instanceId := fmt.Sprintf("%s#%d", entryId, instanceNumber)
	_, ok := wfc.Entries[instanceId]
	for ok {
		lastInstanceId = instanceId

		instanceNumber += 1
		instanceId = fmt.Sprintf("%s#%d", entryId, instanceNumber)
		_, ok = wfc.Entries[instanceId]
	}

	if lastInstanceId == "" {
		return "", fmt.Errorf("cannot find ctxName %s in case", entryId)
	}

	return lastInstanceId, nil
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
		if detail == ReportLogHARRequest && !strings.HasPrefix(e.Comment, InitialRequestHarEntryId) {
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
	instanceId := wfc.ComputeFirstAvailableIndexedHarEntryId(id)
	wfc.Entries[instanceId] = entry
	return nil
}

func (wfc *WfCase) SetHarEntryRequest(id string, req *har.Request, pii config.PersonallyIdentifiableInformation) error {
	const semLogContext = "wf-case::set-har-entry-request"

	instanceId := wfc.ComputeFirstAvailableIndexedHarEntryId(id)
	e, ok := wfc.Entries[instanceId]
	if !ok {
		now := time.Now()
		e = &har.Entry{
			Comment:         instanceId,
			StartedDateTime: now.Format("2006-01-02T15:04:05.999999999Z07:00"),
			StartDateTimeTm: now,
		}
	}
	e.Request = req
	e.PII = har.PersonallyIdentifiableInformation{
		Domain:    pii.Domain,
		AppliesTo: pii.AppliesTo,
	}
	wfc.Entries[instanceId] = e
	return nil
}

func (wfc *WfCase) SetHarEntryResponse(id string, resp *har.Response, pii config.PersonallyIdentifiableInformation) error {

	const semLogContext = "wf-case::set-har-entry-response"

	instanceId, err := wfc.ComputeLastUsedIndexedHarEntryId(id)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	// epData := EndpointData{Id: id, Body: body, Headers: headers, Params: params}
	// wfc.EpInfo[epData.Id] = epData
	e, ok := wfc.Entries[instanceId]
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

	wfc.Entries[instanceId] = e

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

	queryParams := wfc.GetQueryParamsInHarEntry(ctxName.Name)
	req, err := har.NewRequest(method, url, body, httpHeaders, queryParams, nil)
	return req, err
}

func (wfc *WfCase) GetHeaderInHarEntry(ctxName string, hn string) string {

	herEntry, err := wfc.GetHarEntry(ctxName)
	if err != nil {
		return ""
	}

	v := ""
	if strings.HasPrefix(ctxName, InitialRequestHarEntryId) {
		v = herEntry.Request.Headers.GetFirst(hn).Value
	} else {
		v = herEntry.Response.Headers.GetFirst(hn).Value
	}

	return v

}

func (wfc *WfCase) GetHeadersInHarEntry(ctxName string) har.NameValuePairs {

	herEntry, err := wfc.GetHarEntry(ctxName)
	if err != nil {
		return nil
	}

	if strings.HasPrefix(ctxName, InitialRequestHarEntryId) {
		return herEntry.Request.Headers
	} else {
		return herEntry.Response.Headers
	}

	return nil
}

func (wfc *WfCase) GetQueryParamsInHarEntry(ctxName string) har.NameValuePairs {

	herEntry, err := wfc.GetHarEntry(ctxName)
	if err != nil {
		return nil
	}

	if strings.HasPrefix(ctxName, InitialRequestHarEntryId) {
		return herEntry.Request.QueryString
	}

	return nil
}

func (wfc *WfCase) GetBodyInHarEntry(resolverContext HarEntryReference, ignoreNonApplicationJsonResponseContent bool) ([]byte, error) {
	var b []byte
	var err error

	entry, err := wfc.GetHarEntry(resolverContext.Name)
	if err != nil {
		return nil, err
	}

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

	return b, err
}
