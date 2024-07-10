package wfcase

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	utils2 "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/jsonmask"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/templateutil"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"time"
)

type BreadcrumbStep struct {
	Name        string
	Description string
	Err         error
}

type Breadcrumb []BreadcrumbStep

type EndpointData struct {
	Id      string
	Body    []byte
	Headers http.Header
	Params  gin.Params
}

type WfCase struct {
	SymphonyId  string
	Browser     *har.Creator
	Description string
	StartAt     time.Time
	Vars        ProcessVars
	Entries     map[string]*har.Entry
	Dicts       config.Dictionaries
	Refs        config.DataReferences
	Breadcrumb  Breadcrumb
	Span        opentracing.Span
}

func NewWorkflowCase(symphonyId string, version, sha string, descr string, dicts config.Dictionaries, refs config.DataReferences, span opentracing.Span) (*WfCase, error) {

	const semLogContext = "wf-case::new"
	podName := os.Getenv("HOSTNAME")
	if podName == "" {
		log.Warn().Msg(semLogContext + " HOSTNAME env variable not set")
		podName = "localhost"
	}

	browser := &har.Creator{
		Name:    fmt.Sprintf("%s@%s", symphonyId, podName),
		Version: fmt.Sprintf("%s - %s", version, sha),
	}

	// epInfo := make(map[string]EndpointData)
	c := &WfCase{
		SymphonyId: symphonyId,
		Browser:    browser,

		Description: descr,
		StartAt:     time.Now(),
		Entries:     make(map[string]*har.Entry),
		Dicts:       dicts,
		Refs:        refs,
		Span:        span}

	v := ProcessVars(make(map[string]interface{}))
	for fn, fb := range GetFuncMap(c) {
		v[fn] = fb
	}

	v[SymphonyOrchestrationIdProcessVar] = symphonyId
	v[SymphonyOrchestrationDescriptionProcessVar] = descr

	/*
		v["dict"] = func(n string, elems ...string) string {
			s, err := c.Dicts.Map(n, elems...)
			if err != nil {
				log.Error().Err(err).Send()
			}

			return s
		}
	*/
	c.Vars = v
	return c, nil
}

/*
func (wfc *WfCase) AddEndpointData(id string, body []byte, headers http.Header, params gin.Params) error {
	epData := EndpointData{Id: id, Body: body, Headers: headers, Params: params}
	wfc.EpInfo[epData.Id] = epData

	return nil
}
*/

func (wfc *WfCase) AddEndpointRequestData(id string, req *har.Request, pii config.PersonallyIdentifiableInformation) error {
	e, ok := wfc.Entries[id]
	if !ok {
		now := time.Now()
		e = &har.Entry{
			Comment:         id,
			StartedDateTime: time.Now().Format("2006-01-02T15:04:05.999999999Z07:00"),
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

func (wfc *WfCase) GetRequestId() string {
	e, ok := wfc.Entries["request"]
	if !ok || e.Request == nil || len(e.Request.Headers) == 0 {
		return ""
	}

	return e.Request.Headers.GetFirst("requestId").Value
}

/*
func (wfc *WfCase) AddEndpointRequestData(id string, method string, url string, body []byte, headers http.Header, params gin.Params) error {
	// epData := EndpointData{Id: id, Body: body, Headers: headers, Params: params}
	// wfc.EpInfo[epData.Id] = epData
	e, ok := wfc.Entries[id]
	if !ok {
		now := time.Now()
		e = &har.Entry{
			StartedDateTime: time.Now().Format("2006-01-02T15:04:05.000Z"),
			StartDateTimeTm: now,
		}
	}

	var hs []har.NameValuePair
	for n, h := range headers {
		for i := range h {
			hs = append(hs, har.NameValuePair{Name: n, Value: h[i]})
		}
	}

	pars := make([]har.Param, 0)
	for _, h := range params {
		pars = append(pars, har.Param{Name: h.Key, Value: h.Value})

	}

	e.Request = &har.Request{
		Method:      method,
		URL:         url,
		HTTPVersion: "1.1",
		Headers:     hs,
		HeadersSize: -1,
		Cookies:     []har.Cookie{},
		QueryString: []har.NameValuePair{},
		BodySize:    int64(len(body)),
		PostData: &har.PostData{
			MimeType: constants.ContentTypeApplicationJson,
			Data:     body,
			Params:   pars,
		},
	}

	wfc.Entries[id] = e
	return nil
}
*/

func (wfc *WfCase) AddEndpointResponseData(id string, resp *har.Response, pii config.PersonallyIdentifiableInformation) error {

	const semLogContext = "wf-case::add-endpoint-response-data"

	// epData := EndpointData{Id: id, Body: body, Headers: headers, Params: params}
	// wfc.EpInfo[epData.Id] = epData
	e, ok := wfc.Entries[id]
	if !ok {
		now := time.Now()
		e = &har.Entry{
			StartDateTimeTm: now,
			StartedDateTime: time.Now().Format("2006-01-02T15:04:05.999999999Z07:00"),
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

func (wfc *WfCase) AddBreadcrumb(n string, d string, e error) {
	wfc.Breadcrumb = append(wfc.Breadcrumb, BreadcrumbStep{Name: n, Description: d, Err: e})
}

func (wfc *WfCase) GetHeaderFromContext(ctxName string, hn string) string {
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		if ok {
			v := ""
			if ctxName == "request" {
				v = endpointData.Request.Headers.GetFirst(hn).Value
			} else {
				v = endpointData.Response.Headers.GetFirst(hn).Value
			}

			return v
		}
	}

	return ""
}

func (wfc *WfCase) GetResolverForEntry(ctxName string, withVars bool, withTransformationId string, ignoreNonApplicationJsonResponseContent bool) (*ProcessVarResolver, error) {
	var resolver *ProcessVarResolver
	var err error
	if ctxName == "request" {
		resolver, err = wfc.GetResolverForRequestEntry(ctxName, withVars, withTransformationId)
	} else {
		resolver, err = wfc.GetResolverForResponseEntry(ctxName, withVars, withTransformationId, ignoreNonApplicationJsonResponseContent)
	}

	return resolver, err
}

func (wfc *WfCase) GetResolverForRequestEntry(ctxName string, withVars bool, withTransformationId string) (*ProcessVarResolver, error) {

	var err error
	var resolver *ProcessVarResolver
	if endpointData, ok := wfc.Entries[ctxName]; ok {

		opts := []VarResolverOption{WithHeaders(endpointData.Request.Headers)}
		if endpointData.Request.PostData != nil {
			opts = append(opts, WithBody(endpointData.Request.PostData.MimeType, endpointData.Request.PostData.Data, withTransformationId), WithParams(endpointData.Request.PostData.Params))
		}

		if withVars {
			opts = append(opts, WithProcessVars(wfc.Vars))
		}
		resolver, err = NewProcessVarResolver(opts...)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("cannot find ctxName %s in case", ctxName)
	}

	return resolver, nil
}

func (wfc *WfCase) GetRequestEntry(ctxName string) (*har.Entry, error) {
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		return endpointData, nil
	}
	return nil, fmt.Errorf("cannot find ctxName %s in case", ctxName)
}

func (wfc *WfCase) GetResolverForResponseEntry(ctxName string, withVars bool, withTransformationId string, ignoreNonApplicationJsonContent bool) (*ProcessVarResolver, error) {

	var err error
	var resolver *ProcessVarResolver
	if endpointData, ok := wfc.Entries[ctxName]; ok {
		opts := []VarResolverOption{WithHeaders(endpointData.Response.Headers)}
		if endpointData.Response.Content != nil && len(endpointData.Response.Content.Data) > 0 {
			// This condition should not consider the body if is not application json and the ignore flag has been set to true
			if strings.HasPrefix(endpointData.Response.Content.MimeType, constants.ContentTypeApplicationJson) || !ignoreNonApplicationJsonContent {
				opts = append(opts, WithBody(endpointData.Response.Content.MimeType, endpointData.Response.Content.Data, withTransformationId))
			} else {
				log.Debug().Str("content-type", endpointData.Response.Content.MimeType).Msg("ignoring body")
			}
		}
		if withVars {
			opts = append(opts, WithProcessVars(wfc.Vars))
		}
		resolver, err = NewProcessVarResolver(opts...)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("cannot find ctxName %s in case", ctxName)
	}

	return resolver, nil
}

func (wfc *WfCase) SetVars(ctxName string, vars []config.ProcessVar, transformationId string, ignoreNonApplicationJsonResponseContent bool) error {

	if len(vars) == 0 {
		return nil
	}

	resolver, err := wfc.GetResolverForEntry(ctxName, true, transformationId, ignoreNonApplicationJsonResponseContent)

	if err != nil {
		return err
	}

	for _, v := range vars {

		boolGuard := true
		if v.Guard != "" {
			boolGuard, err = wfc.Vars.BoolEval(v.Guard)
		}

		if boolGuard && err == nil {
			err = wfc.Vars.Set(v.Name, v.Value, resolver)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (wfc *WfCase) ResolveStrings(ctxName string, expr []string, transformationId string, ignoreNonApplicationJsonResponseContent bool) ([]string, error) {

	resolver, err := wfc.GetResolverForEntry(ctxName, true, transformationId, ignoreNonApplicationJsonResponseContent)
	if err != nil {
		return nil, err
	}

	var resolved []string
	for _, s := range expr {
		val, _, err := varResolver.ResolveVariables(s, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, val)
	}

	return resolved, nil
}

func (wfc *WfCase) ShowVars(sorted bool) {

	var varNames []string
	if sorted {
		log.Warn().Msg("please disable sorting of process variables")
		for n, _ := range wfc.Vars {
			varNames = append(varNames, n)
		}

		sort.Strings(varNames)
		for _, n := range varNames {
			i := wfc.Vars[n]
			if reflect.ValueOf(i).Kind() != reflect.Func {
				log.Trace().Str("name", n).Interface("value", i).Msg("case variable")
			}
		}
	} else {
		for n, v := range wfc.Vars {
			if reflect.ValueOf(v).Kind() != reflect.Func {
				log.Trace().Str("name", n).Interface("value", v).Msg("case variable")
			}
		}
	}
}

func (wfc *WfCase) ShowBreadcrumb() {
	for stepNumber, v := range wfc.Breadcrumb {
		log.Trace().Int("at", stepNumber).Str("name", v.Name).Msg("breadcrumb")
		if v.Err != nil {
			log.Trace().Str("with-err", v.Err.Error()).Int("at", stepNumber).Str("name", v.Name).Msg("breadcrumb")
		}
	}
}

func (wfc *WfCase) BooleanEvalProcessVars(varExpressions []string) (int, error) {
	return wfc.Vars.Eval(varExpressions, ExactlyOne)
}

func (wfc *WfCase) EvalExpression(varExpression string) bool {
	_, err := wfc.Vars.Eval([]string{varExpression}, ExactlyOne)
	return err == nil
}

func (wfc *WfCase) getTemplateFunctions() template.FuncMap {

	fMap := template.FuncMap(GetFuncMap(wfc))
	/* {
		"dict": func(dictName string, elems ...string) string {
			s, err := wfc.Dicts.Map(dictName, elems...)
			if err != nil {
				log.Error().Err(err).Send()
			}
			return s
		},
		"now": func(format string) string {
			return time.Now().Format(format)
		},
	}*/

	return fMap
}

func (wfc *WfCase) ProcessTemplate(tmpl string) ([]byte, error) {

	ti := []templateutil.Info{
		{
			Name:    "body",
			Content: tmpl,
		},
	}

	pkgTemplate, err := templateutil.Parse(ti, wfc.getTemplateFunctions())
	if err != nil {
		return nil, err
	}

	resp, err := templateutil.Process(pkgTemplate, wfc.Vars, false)
	if err != nil {
		return nil, err
	}

	return resp, nil
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
				Version: utils2.GetVersion(),
			},
			Browser: wfc.Browser,
			Comment: wfc.SymphonyId,
		},
	}

	for n, e := range wfc.Entries {

		incl := true
		if detail == ReportLogHARRequest && e.Comment != "request" {
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

/*
func maskEnrty(e *har.Entry, jm *jsonmask.JsonMask) error {

	if jm != nil {
		if e.PII.ShouldMaskRequest() {

		}

		if e.PII.ShouldMaskResponse() {

		}
	}

	return nil
}
*/
