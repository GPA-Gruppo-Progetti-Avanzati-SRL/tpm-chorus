package kafkactivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"net/http"
	"strings"
	"time"
)

const (
	MessageKeyHeaderName = "X-Kafka-Key"
)

const (
	MetricIdActivityType           = "type"
	MetricIdActivityName           = "name"
	MetricIdEndpointDefinitionPath = "endpoint"
	MetricIdEndpointId             = "endpoint-id"
	MetricIdEndpointName           = "endpoint-name"
	MetricIdStatusCode             = "status-code"
	MetricIdMethod                 = "http-method"
	MetricIdBrokerName             = "broker-name"
	MetricIdTopicName              = "topic-name"
)

type Producer struct {
	Id          string
	Name        string
	Description string
	Definition  *config.ProducerDefinition
	PII         config.PersonallyIdentifiableInformation
}

func (p *Producer) getRequestSpan(parentSpan opentracing.Span) (opentracing.Span, bool) {
	if p.Definition.TraceOpName != "" {
		var span opentracing.Span
		if parentSpan != nil {
			parentSpanCtx := parentSpan.Context()
			span = opentracing.StartSpan(p.Definition.TraceOpName, opentracing.ChildOf(parentSpanCtx))
		} else {
			span = opentracing.StartSpan(p.Definition.TraceOpName)
		}

		return span, true
	}

	return parentSpan, false
}

type KafkaActivity struct {
	executable.Activity
	BrokerName string
	Producers  []Producer
}

func NewKafkaActivity(item config.Configurable, refs config.DataReferences) (*KafkaActivity, error) {

	const semLogContext = "kafka-activity::new"
	ea := &KafkaActivity{}
	ea.Cfg = item
	ea.Refs = refs

	tcfg := item.(*config.KafkaActivity)
	ea.BrokerName = tcfg.BrokerName
	for _, epcfg := range tcfg.Producers {

		epCfgDef, _ := refs.Find(epcfg.Definition)
		if len(epCfgDef) == 0 {
			return nil, fmt.Errorf(semLogContext+" cannot find producer (%s:%s) definition from %s", epcfg.Id, epcfg.Name, epcfg.Definition)
		}

		epDef := config.ProducerDefinition{}
		err := yaml.Unmarshal(epCfgDef, &epDef)
		if err != nil {
			return nil, err
		}

		if epDef.Body.ExternalValue != "" && !refs.IsPresent(epDef.Body.ExternalValue) {
			return nil, fmt.Errorf(semLogContext+" cannot find producer (%s:%s) body reference from %s", epcfg.Id, epcfg.Name, epDef.Body.ExternalValue)
		}

		ep := Producer{Id: epcfg.Id, Name: epcfg.Name, Description: epcfg.Description, Definition: &epDef, PII: epcfg.PII}
		ea.Producers = append(ea.Producers, ep)
	}

	return ea, nil
}

func (a *KafkaActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.KafkaActivityType) + "::execute"

	var err error
	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.KafkaActivityType)).Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	cfg, ok := a.Cfg.(*config.KafkaActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.KafkaActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	err = cfg.WfCaseDeadlineExceeded(wfc.RequestTiming, wfc.RequestDeadline)
	if err != nil {
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	activityBegin := time.Now()
	defer func(begin time.Time) {
		wfc.RequestTiming += time.Since(begin)
		log.Info().Str(constants.SemLogActivity, a.Name()).Float64("wfc-timing.s", wfc.RequestTiming.Seconds()).Float64("deadline.s", wfc.RequestDeadline.Seconds()).Msg(semLogContext + " - wfc timing")
	}(activityBegin)

	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}
	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

	//if len(cfg.ProcessVars) > 0 {
	err = wfc.SetVars(expressionCtx, cfg.ProcessVars, "", false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}
	//}

	for _, ep := range a.Producers {

		beginOf := time.Now()
		metricsLabels := a.MetricsLabels(a.BrokerName, ep)

		req, err := a.newRequestDefinition(wfc, ep)
		if err != nil {
			wfc.AddBreadcrumb(ep.Id, ep.Description, err)
			metricsLabels[MetricIdStatusCode] = "500"
			_ = a.SetMetrics(beginOf, metricsLabels)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(ep.Name), smperror.WithStep(ep.Id), smperror.WithCode("HTTP"), smperror.WithErrorMessage(err.Error()))
		}

		// Note: owned span doesn't finish if exits in between...
		span, owned := ep.getRequestSpan(wfc.Span)

		if span != nil {
			m := make(map[string]string)
			opentracing.GlobalTracer().Inject(
				span.Context(),
				opentracing.TextMap,
				opentracing.TextMapCarrier(m))
			for n, v := range m {
				req.Headers = append(req.Headers, har.NameValuePair{Name: n, Value: v})
			}
		}

		_ = wfc.SetHarEntryRequest(ep.Id, req, ep.PII)

		entry, err := a.Produce(wfc, ep, req)
		var resp *har.Response
		if entry != nil {
			resp = entry.Response
		}
		_ = wfc.SetHarEntryResponse(ep.Id, resp, ep.PII)

		if owned {
			span.Finish()
		}

		metricsLabels[MetricIdStatusCode] = fmt.Sprint(resp.Status)

		remappedStatusCode, err := a.ProcessResponseActionByStatusCode(
			resp.Status, a.Name(), util.StringCoalesce(ep.Id, ep.Name), wfc, nil, wfcase.HarEntryReference{Name: ep.Id, UseResponse: true}, ep.Definition.OnResponseActions, false)
		if remappedStatusCode > 0 {
			metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
		}
		if err != nil {
			wfc.AddBreadcrumb(ep.Id, ep.Description, err)
			_ = a.SetMetrics(beginOf, metricsLabels)
			return err
		}

		/*
			actNdx := ep.findProducerResponseAction(resp.Status)
			if actNdx >= 0 {
				remappedStatusCode, err := a.processProducerResponseAction(wfc, a.Name(), ep, actNdx, resp)
				if remappedStatusCode != 0 {
					metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
				}
				if err != nil {
					wfc.AddBreadcrumb(ep.Id, ep.Description, err)
					_ = a.SetMetrics(beginOf, metricsLabels)
					return err
				}
			}
		*/

		_ = a.SetMetrics(beginOf, metricsLabels)
		wfc.AddBreadcrumb(ep.Id, ep.Description, nil)
	}

	return nil
}

/*
func (a *KafkaActivity) processProducerResponseAction(wfc *wfcase.WfCase, activityName string, ep Producer, actionIndex int, resp *har.Response) (int, error) {
	const semLogContext = "kafka-activity::process-producer-response-action"
	act := ep.Definition.OnResponseActions[actionIndex]

	expressionCtx, err := wfc.ResolveExpressionContextName(ep.Id)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorMessage(err.Error()), smperror.WithDescription(err.Error()))
	}

	ignoreNonJSONResponseContent := false
	if len(act.ProcessVars) > 0 {
		err := wfc.SetVars(expressionCtx, act.ProcessVars, "", ignoreNonJSONResponseContent)
		if err != nil {
			log.Error().Err(err).Str("ctx", ep.Id).Str("request-id", wfc.GetRequestId()).Msg("processResponseAction: error in setting variables")
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(ep.Name), smperror.WithCode("500"), smperror.WithErrorMessage("error processing response body"), smperror.WithDescription(err.Error()))
		}
	}

	if ndx := a.chooseProducerError(wfc, act.Errors); ndx >= 0 {

		e := act.Errors[ndx]
		ambit := e.Ambit
		if ambit == "" {
			ambit = activityName
		}

		step := e.Step
		if step == "" {
			step = ep.Name
		}
		if step == "" {
			step = ep.Id
		}

		statusCode := int(resp.Status)
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}

		m, err := wfc.ResolveStrings(expressionCtx, []string{e.Code, e.Message, e.Description, step}, "", ignoreNonJSONResponseContent)
		if err != nil {
			log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
		}
		return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]))
	}

	return 0, nil
}

func (a *KafkaActivity) chooseProducerError(wfc *wfcase.WfCase, errors []config.ErrorInfo) int {
	for i, e := range errors {
		if e.Guard == "" {
			return i
		}

		if wfc.EvalExpression(e.Guard) {
			return i
		}
	}

	return -1
}

func (ep *Producer) findProducerResponseAction(statusCode int) int {

	matchedAction := -1
	defaultAction := -1
	for ndx, act := range ep.Definition.OnResponseActions {
		if act.StatusCode == statusCode {
			matchedAction = ndx
			break
		}

		if act.StatusCode == -1 {
			defaultAction = ndx
		}
	}

	if matchedAction < 0 && defaultAction >= 0 {
		matchedAction = defaultAction
	}

	return matchedAction
}
*/

func (a *KafkaActivity) Produce(wfc *wfcase.WfCase, ep Producer, reqDef *har.Request) (*har.Entry, error) {

	const semLogContext = "kafka-activity::produce"
	lks, err := kafkalks.GetKafkaLinkedService(a.BrokerName)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	producer, err := lks.NewSharedProducer(context.Background(), false)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	now := time.Now()
	e := &har.Entry{
		StartedDateTime: now.Format(time.RFC3339Nano),
		StartDateTimeTm: now,
		Request:         reqDef,
	}

	var msgKey string
	var msgHeaders map[string]string
	if len(reqDef.Headers) > 0 {
		msgHeaders = make(map[string]string)
		for _, h := range reqDef.Headers {
			msgHeaders[h.Name] = h.Value
			if h.Name == MessageKeyHeaderName {
				msgKey = h.Value
			}
		}
	}

	sc, resp := producer.Produce2Topic(ep.Definition.TopicName, []byte(msgKey), reqDef.PostData.Data, msgHeaders, nil)

	b, err := json.Marshal(resp)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	responseHeaders := []har.NameValuePair{{Name: "Content-Type", Value: constants.ContentTypeApplicationJson}, {Name: "Content-Length", Value: fmt.Sprint(len(b))}}
	r := &har.Response{
		Status:      sc,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(sc),
		Headers:     responseHeaders,
		HeadersSize: -1,
		BodySize:    int64(len(b)),
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeApplicationJson,
			Size:     int64(len(b)),
			Data:     b,
		},
	}

	if e.StartedDateTime != "" {
		elapsed := time.Since(e.StartDateTimeTm)
		e.Time = float64(elapsed.Milliseconds())
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

	e.Response = r

	log.Trace().Int("status-code", r.Status).Int("num-headers", len(r.Headers)).Int64("content-length", r.BodySize).Msg(semLogContext + " message produced")

	ct := r.Content.MimeType
	if err == nil && !strings.HasPrefix(ct, constants.ContentTypeApplicationJson) && r.Status != 200 && r.BodySize > 0 {
		// err = fmt.Errorf("%s", string(resp.Content.Data))
		log.Warn().Str("content-type", ct).Msg("content is not the usual " + constants.ContentTypeApplicationJson)
	}

	return e, err
}

func (a *KafkaActivity) newRequestDefinition(wfc *wfcase.WfCase, ep Producer) (*har.Request, error) {

	const semLogContext = "kafka-activity::new-request-definition"
	// note the ignoreNonApplicationJsonResponseContent has been set to false since it doesn't apply to the request processing
	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		return nil, err
	}

	resolver, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", false)
	if err != nil {
		return nil, err
	}

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithScheme("activity")

	ub.WithHostname(fmt.Sprintf("%s", a.BrokerName))

	s, _, err := varResolver.ResolveVariables(ep.Definition.TopicName, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}
	ub.WithPath(fmt.Sprintf("/%s/%s/topics/%s", string(config.KafkaActivityType), a.Name(), s))
	opts = append(opts, har.WithMethod(http.MethodPost))
	opts = append(opts, har.WithUrl(ub.Url()))

	for _, h := range ep.Definition.Headers {
		r, _, err := varResolver.ResolveVariables(h.Value, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}
		opts = append(opts, har.WithHeader(har.NameValuePair{Name: h.Name, Value: r}))
	}

	if !ep.Definition.Body.IsZero() {
		opt, err := a.newRequestDefinitionMessageBody(wfc, ep, resolver)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	} else {
		err = errors.New(" body of activity cannot be empty")
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	opt, err := a.newRequestDefinitionMessageKey(wfc, ep, resolver)
	if err != nil {
		return nil, err
	}
	opts = append(opts, opt)

	req := har.Request{
		HTTPVersion: "1.1",
		Cookies:     []har.Cookie{},
		QueryString: []har.NameValuePair{},
		HeadersSize: -1,
		Headers:     []har.NameValuePair{},
		BodySize:    -1,
	}
	for _, o := range opts {
		o(&req)
	}

	return &req, nil
}

func (a *KafkaActivity) newRequestDefinitionMessageBody(wfc *wfcase.WfCase, ep Producer, resolver *wfexpressions.Evaluator) (har.RequestOption, error) {

	var bodyContent []byte
	if ep.Definition.Body.ExternalValue != "" {
		bodyContent, _ = a.Refs.Find(ep.Definition.Body.ExternalValue)
	} else {
		bodyContent = []byte(ep.Definition.Body.Value)
	}

	s, _, err := varResolver.ResolveVariables(string(bodyContent), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}

	if ep.Definition.Body.Type == "simple" {
		return har.WithBody([]byte(s)), nil
	}

	b, err := wfc.ProcessTemplate(s)
	if err != nil {
		return nil, err
	}

	return har.WithBody(b), nil

}

func (a *KafkaActivity) newRequestDefinitionMessageKey(wfc *wfcase.WfCase, ep Producer, resolver *wfexpressions.Evaluator) (har.RequestOption, error) {

	const semLogContext = "kafka-activity::new-message-key"

	var s string
	var err error
	if ep.Definition.Key != "" {
		// messageKey, _ := a.Refs.Find(ep.Definition.Key)
		messageKey := ep.Definition.Key
		s, _, err = varResolver.ResolveVariables(string(messageKey), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}
	} else {
		s = "kafka-activity"
		log.Warn().Msgf("message key not specified... defaulting to '%s'", s)
	}

	return har.WithHeader(har.NameValuePair{Name: MessageKeyHeaderName, Value: s}), nil
}

func (a *KafkaActivity) MetricsLabels(brokerName string, p Producer) prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType:           string(a.Cfg.Type()),
		MetricIdActivityName:           a.Name(),
		MetricIdEndpointDefinitionPath: fmt.Sprintf("%s://%s.tpm-symphony/topics/%s", "kafka", brokerName, p.Definition.TopicName),
		MetricIdTopicName:              p.Definition.TopicName,
		MetricIdEndpointId:             p.Id,
		MetricIdEndpointName:           p.Name,
		MetricIdBrokerName:             brokerName,
		MetricIdStatusCode:             "-1",
		MetricIdMethod:                 http.MethodPost,
	}

	return metricsLabels
}
