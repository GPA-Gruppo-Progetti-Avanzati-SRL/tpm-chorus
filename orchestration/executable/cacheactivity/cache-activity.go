package cacheactivity

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cacheoperation"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type CacheActivity struct {
	executable.Activity
	definition config.CacheActivityDefinition
}

func NewCacheActivity(item config.Configurable, refs config.DataReferences) (*CacheActivity, error) {
	const semLogContext = string(config.CacheActivityType) + "::new"
	var err error

	ma := &CacheActivity{}
	ma.Cfg = item
	ma.Refs = refs

	maCfg := item.(*config.CacheActivity)
	ma.definition, err = config.UnmarshalCacheActivityDefinition(maCfg.Name(), maCfg.Definition, refs)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return ma, nil
}

func (a *CacheActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.CacheActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.CacheActivityType)).Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.CacheActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.CacheActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	err = tcfg.WfCaseDeadlineExceeded(wfc.RequestTiming, wfc.RequestDeadline)
	if err != nil {
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	activityBegin := time.Now()
	defer func(begin time.Time) {
		wfc.RequestTiming += time.Since(begin)
		log.Info().Str(constants.SemLogActivity, a.Name()).Float64("wfc-timing.s", wfc.RequestTiming.Seconds()).Float64("deadline.s", wfc.RequestDeadline.Seconds()).Msg(semLogContext + " - wfc timing")
	}(activityBegin)

	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}
	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

	if len(tcfg.ProcessVars) > 0 {
		err := wfc.SetVars(expressionCtx, tcfg.ProcessVars, "", false)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func() { a.SetMetrics(beginOf, metricsLabels) }()

	evaluator, err := a.GetEvaluator(wfc)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	cacheCfg, err := a.resolveCacheConfig(wfc, evaluator, config.CacheConfig{
		Key:              a.definition.Key,
		Namespace:        a.definition.Namespace,
		Ttl:              a.definition.Ttl,
		LinkedServiceRef: a.definition.LinkedServiceRef,
	}, a.Refs)

	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	var harResponse *har.Response
	switch a.definition.Operation {
	case config.CacheOperationGet:
		harResponse, err = a.executeGet(wfc, cacheCfg)
	case config.CacheOperationSet:
		harResponse, err = a.executeSet(wfc, expressionCtx, cacheCfg)
	default:
		err = errors.New("unknown operation")
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
	}

	if harResponse != nil {
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		}
		_ = wfc.SetHarEntryResponse(a.Name(), harResponse, config.PersonallyIdentifiableInformation{})
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(harResponse.Status)
	} else {
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			metricsLabels[MetricIdStatusCode] = "500"
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	remappedStatusCode, err := a.ProcessResponseActionByStatusCode(harResponse.Status, a.Name(), a.Name(), wfc, nil, wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}, a.definition.OnResponseActions, true)
	if remappedStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
	}
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		_ = a.SetMetrics(beginOf, metricsLabels)
		return err
	}

	// _ = a.SetMetrics(beginOf, metricsLabels)
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return err
}

func (a *CacheActivity) executeGet(wfc *wfcase.WfCase, cacheConfig config.CacheConfig) (*har.Response, error) {
	cacheHarEntry, err := cacheoperation.Get(
		cacheConfig.LinkedServiceRef,
		a.Name(),
		cacheConfig.Key,
		constants.ContentTypeApplicationJson,
		cachelks.WithNamespace(cacheConfig.Namespace), cachelks.WithHarPath(fmt.Sprintf("/%s/%s/%s", string(config.MongoActivityType), string(a.definition.Operation), a.Name())))

	if cacheHarEntry != nil {
		_ = wfc.SetHarEntry(a.Name(), cacheHarEntry)
		return cacheHarEntry.Response, err
	}

	return nil, err
}

func (a *CacheActivity) executeSet(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference, cacheConfig config.CacheConfig) (*har.Response, error) {
	const semLogContext = "cache-activity::execute-set"

	now := time.Now()

	req, err := a.newRequestDefinition(wfc, expressionCtx) // TODO calcolare lo statement
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	st := http.StatusOK
	err = cacheoperation.Set(cacheConfig.LinkedServiceRef, cacheConfig.Key, req.PostData.Data, cachelks.WithNamespace(cacheConfig.Namespace), cachelks.WithTTTL(cacheConfig.Ttl))
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		st = http.StatusInternalServerError
	}

	var r *har.Response
	r = &har.Response{
		Status:      st,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(st),
		HeadersSize: -1,
		BodySize:    int64(len(http.StatusText(st))),
		Headers:     []har.NameValuePair{},
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeTextPlain,
			Size:     int64(len(http.StatusText(st))),
			Data:     []byte(http.StatusText(st)),
		},
	}

	elapsed := float64(time.Since(now).Milliseconds())
	harEntry := &har.Entry{
		Comment:         a.Name(),
		StartedDateTime: now.Format("2006-01-02T15:04:05.999999999Z07:00"),
		StartDateTimeTm: now,
		Request:         req,
		Response:        r,
		Time:            elapsed,
		Timings: &har.Timings{
			Blocked: -1,
			DNS:     -1,
			Connect: -1,
			Send:    -1,
			Wait:    elapsed,
			Receive: -1,
			Ssl:     -1,
		},
	}

	_ = wfc.SetHarEntry(a.Name(), harEntry)
	return r, nil
}

func (a *CacheActivity) resolveCacheConfig(wfc *wfcase.WfCase, resolver *wfexpressions.Evaluator, cacheConfig config.CacheConfig, refs config.DataReferences) (config.CacheConfig, error) {
	cfg := cacheConfig
	if refs.IsPresent(cacheConfig.Key) {
		if key, ok := refs.Find(cacheConfig.Key); ok {
			cfg.Key = string(key)
		}
	}

	s, _, err := varResolver.ResolveVariables(cfg.Key, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return cfg, err
	}

	b1, err := wfc.ProcessTemplate(s)
	if err != nil {
		return cfg, err
	}

	cfg.Key = string(b1)
	return cfg, err
}

func (a *CacheActivity) newRequestDefinition(wfc *wfcase.WfCase, expressionCtx wfcase.HarEntryReference) (*har.Request, error) {

	const semLogContext = "cache-activity::new-request-definition"

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(0000)
	ub.WithScheme("activity")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s", string(config.CacheActivityType), a.Name()))
	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))

	b, _, err := wfc.GetBodyInHarEntry(expressionCtx, true)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}
	opts = append(opts, har.WithBody(b))

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

const (
	MetricIdActivityType = "type"
	MetricIdActivityName = "name"
	MetricIdOpType       = "op-type"
	MetricIdStatusCode   = "status-code"
)

func (a *CacheActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}
