package wfcase

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

const (
	ContentTypeHeaderName                      = "content-type"
	RhapsodyPipelineIdProcessVar               = "rhp_pipeline_id"
	RhapsodyPipelineIdHeaderName               = "X-Rhp-Pipeline-Id"
	RhapsodyRequestIdHeaderName                = "X-Rhp-Request-Id"
	RhapsodyPipelineDescrProcessVar            = "rhp_pipeline_descr"
	SymphonyOrchestrationIdProcessVar          = "smp_orchestration_id"
	SymphonyOrchestrationDescriptionProcessVar = "smp_orchestration_descr"
)

/*
type HttpParam struct {
	Key   string
	Value string
}

type EndpointData struct {
	Id      string
	Body    []byte
	Headers http.Header
	Params  []HttpParam
}
*/

type WfCase struct {
	Id          string
	RequestId   string
	Browser     *har.Creator
	Description string
	StartAt     time.Time
	Vars        wfexpressions.ProcessVars
	Entries     map[string]*har.Entry
	Dicts       config.Dictionaries
	Refs        config.DataReferences
	Breadcrumb  Breadcrumb
	Span        opentracing.Span

	ExpressionEvaluator *wfexpressions.Evaluator
}

func NewWorkflowCase(id string, version, sha string, descr string, dicts config.Dictionaries, refs config.DataReferences, systemVars map[string]interface{}, span opentracing.Span) (*WfCase, error) {

	const semLogContext = "wf-case::new"
	podName := os.Getenv("HOSTNAME")
	if podName == "" {
		log.Warn().Msg(semLogContext + " HOSTNAME env variable not set")
		podName = "localhost"
	}

	browser := &har.Creator{
		Name:    fmt.Sprintf("%s@%s", id, podName),
		Version: fmt.Sprintf("%s - %s", version, sha),
	}

	// epInfo := make(map[string]EndpointData)
	c := &WfCase{
		Id:      id,
		Browser: browser,

		Description: descr,
		StartAt:     time.Now(),
		Entries:     make(map[string]*har.Entry),
		Dicts:       dicts,
		Refs:        refs,
		Span:        span}

	v := wfexpressions.ProcessVars(make(map[string]interface{}))
	for fn, fb := range GetFuncMap(c) {
		v[fn] = fb
	}

	for n, val := range systemVars {
		v[n] = val
		log.Info().Str("name", n).Interface("value", val).Msg(semLogContext + " - setting system variable")
	}

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

func (wfc *WfCase) NewChild(expressionCtx HarEntryReference, id string, version, sha string, descr string, dicts config.Dictionaries, refs config.DataReferences, vars []config.ProcessVar, body []byte, span opentracing.Span) (*WfCase, error) {
	const semLogContext = "wf-case::new-child"
	childWfc, err := NewWorkflowCase(id, version, sha, descr, dicts, refs, nil, span)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	err = childWfc.SetVarsFromCase(wfc, expressionCtx, vars, "", false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	req, err := wfc.NewHarRequestFromHarEntryReference(expressionCtx, "POST", fmt.Sprintf("activity://localhost/%s/%s", config.NestedOrchestrationActivityType, id), body)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return childWfc, err
	}

	err = childWfc.SetHarEntryRequest("request", req, config.PersonallyIdentifiableInformation{})
	if err != nil {
		log.Error().Err(err)
		return childWfc, err
	}

	return childWfc, nil
}

/*
func (wfc *WfCase) AddEndpointData(id string, body []byte, headers http.Header, params gin.Params) error {
	epData := EndpointData{Id: id, Body: body, Headers: headers, Params: params}
	wfc.EpInfo[epData.Id] = epData

	return nil
}
*/

func (wfc *WfCase) GetRequestId() string {
	const semLogContext = "wf-case::get-request-id"

	// This code is to move the setting of requestId outside, not any more dependent from constant
	if wfc.RequestId == "" {
		log.Warn().Msg(semLogContext + " request-id has not been set in case")
	}

	e, ok := wfc.Entries[InitialRequestHarEntryId]
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
/*
func (wfc *WfCase) ShowVars(sorted bool) {

	wfc.Vars.ShowVars(sorted)


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
*/
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
