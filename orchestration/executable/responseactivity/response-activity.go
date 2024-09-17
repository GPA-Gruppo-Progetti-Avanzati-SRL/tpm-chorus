package responseactivity

import (
	"context"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/redislks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/linkedservices"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"

	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/rs/zerolog/log"
)

const (
	MetricIdActivityType = "type"
	MetricIdActivityName = "name"
	MetricIdStatusCode   = "status-code"
	MetricIdCached       = "cached"
	MetricIdCacheMiss    = "cache-miss"
)

type templateData struct {
	simpleResponse      []byte
	onCacheMissResponse []byte
}

type ResponseActivity struct {
	executable.Activity
	// templates map[string]templateData
}

func NewResponseActivity(item config.Configurable, refs config.DataReferences) (*ResponseActivity, error) {

	const semLogContext = "response-activity::new"
	a := &ResponseActivity{}
	a.Cfg = item
	a.Refs = refs
	tcfg := item.(*config.ResponseActivity)

	if len(tcfg.Responses) == 0 {
		err := errors.New("response activity missing response configuration")
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	// a.templates = make(map[string]templateData)
	for _, cfgResp := range tcfg.Responses {
		t := templateData{}

		if cfgResp.RefSimpleResponse != "" {
			t.simpleResponse, _ = refs.Find(cfgResp.RefSimpleResponse)
			if len(t.simpleResponse) == 0 {
				err := errors.New("simple response cannot be read")
				log.Error().Str("ref-simple-response", cfgResp.RefSimpleResponse).Err(err).Msg(semLogContext)
				return nil, err
			}
		}

		if cfgResp.Cache.OnCacheMiss.RefCacheMissResponse != "" {
			t.onCacheMissResponse, _ = refs.Find(cfgResp.Cache.OnCacheMiss.RefCacheMissResponse)
			if len(t.onCacheMissResponse) == 0 {
				err := errors.New("on cache miss response cannot be read")
				log.Error().Str("ref-simple-response", cfgResp.Cache.OnCacheMiss.RefCacheMissResponse).Err(err).Msg(semLogContext)
				return nil, err
			}
		}

		// a.templates[cfgResp.Id] = t
	}

	return a, nil
}

func (a *ResponseActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.ResponseActivityType) + "::execute"

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", "response").Msg("start activity")
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)

	expressionCtx, err := wfc.ResolveExpressionContextName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}
	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext + " start")

	cfg, ok := a.Cfg.(*config.ResponseActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.RequestActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	//if len(cfg.ProcessVars) > 0 {
	err = wfc.SetVars(expressionCtx, cfg.ProcessVars, "", false)
	if err != nil {
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}
	//}

	log.Trace().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")
	return nil
}

func (a *ResponseActivity) IsValid() bool {

	const semLogContext = string(config.ResponseActivityType) + "::is-valid"
	const semLogResponseId = "response-id"
	b := a.Activity.IsValid()
	if b {
		tcfg := a.Cfg.(*config.ResponseActivity)
		for _, cfgResp := range tcfg.Responses {
			simpleResponse, _ := a.Refs.Find(cfgResp.RefSimpleResponse) // a.templates[cfgResp.Id]
			if cfgResp.RefSimpleResponse != "" && simpleResponse == nil {
				log.Error().Str(semLogResponseId, cfgResp.Id).Str(constants.SemLogActivityName, a.Name()).Msg(semLogContext + " response activity missing simple response")
				b = false
			}
		}
	}

	return b
}

func (a *ResponseActivity) ResponseJSON(wfc *wfcase.WfCase) (*har.Response, error) {

	const semLogContext = string(config.ResponseActivityType) + "::response-json"

	var err error
	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return nil, err
	}

	tcfg := a.Cfg.(*config.ResponseActivity)
	expressionCtx, _ := wfc.ResolveExpressionContextName(a.Cfg.ExpressionContextNameStringReference())
	resolver, err := wfc.GetResolverByContext(expressionCtx, true, "", false)
	if err != nil {
		return nil, err
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func(start time.Time) {
		a.SetMetrics(start, metricsLabels)
	}(beginOf)

	respNdx := a.selectResponse(tcfg, wfc)
	if respNdx < 0 {
		err = errors.New("unable to select response")
		log.Error().Err(err).Str("name", a.Name()).Msg(semLogContext)
		return nil, err
	}

	r := tcfg.Responses[respNdx]
	statusCode := http.StatusOK
	statusText := http.StatusText(http.StatusOK)
	if r.StatusCode > 0 {
		statusCode = r.StatusCode
		statusText = http.StatusText(r.StatusCode)
	}

	var hs []har.NameValuePair
	hs, err = a.computeHeaders(r.Headers, resolver)
	if err != nil {
		return nil, err
	}

	var body []byte

	body, respType, err := a.handleResponseCache(&r, resolver, r.Cache)
	if err != nil {
		return nil, err
	}

	switch respType {
	case ResponseNotCached:
		simpleResponse, _ := a.Refs.Find(r.RefSimpleResponse)
		body, err = a.computeBody(wfc, simpleResponse, resolver)
		if err != nil {
			return nil, err
		}

		if r.Cache.Mode == config.CacheModeSet && r.Cache.Key != "" {
			err = a.setCachedResponse(resolver, r.Cache.BrokerName, r.Cache.Key, body)
			if err != nil {
				log.Error().Err(err).Str(constants.SemLogCacheKey, r.Cache.Key).Msg(semLogContext + " set cache key error")
			}
		}
	case ResponseCacheHit:
		metricsLabels[MetricIdCached] = "Y"
	case ResponseCacheMiss:
		metricsLabels[MetricIdCached] = "Y"
		metricsLabels[MetricIdCacheMiss] = "Y"
		if r.Cache.OnCacheMiss.StatusCode > 0 {
			statusCode = r.Cache.OnCacheMiss.StatusCode
		}

		cacheMissResponse, _ := a.Refs.Find(r.Cache.OnCacheMiss.RefCacheMissResponse)
		body, err = a.computeBody(wfc, cacheMissResponse, resolver)
		if err != nil {
			return nil, err
		}
	default:
		log.Error().Err(err).Int("cache-response-type", respType).Msg(semLogContext + " default unexpected switch branch taken")
	}

	/*
		if r.Cache.Mode == config.CacheModeGet && r.Cache.Key != "" {
			metricsLabels[MetricIdCached] = "Y"
			body, err = a.handleResponseCache(wfc, &r, resolver, r.Cache.Key)
			if err != nil {
				return nil, err
			}

			// Handle the cache miss
			if body == nil {
				metricsLabels[MetricIdCacheMiss] = "Y"
				if r.Cache.OnCacheMiss.StatusCode > 0 {
					statusCode = r.Cache.OnCacheMiss.StatusCode
				}

				cacheMissResponse, _ := a.Refs.Find(r.Cache.OnCacheMiss.RefCacheMissResponse)
				body, err = a.computeBody(wfc, cacheMissResponse, resolver)
				if err != nil {
					return nil, err
				}
			}
		} else {
			simpleResponse, _ := a.Refs.Find(r.RefSimpleResponse)
			body, err = a.computeBody(wfc, simpleResponse, resolver)
			if err != nil {
				return nil, err
			}

			if r.Cache.Mode == config.CacheModeSet && r.Cache.Key != "" {
				err = a.setCachedResponse(resolver, r.Cache.Key, body)
				if err != nil {
					log.Error().Err(err).Str(constants.SemLogCacheKey, r.Cache.Key).Msg("redis set key error")
				}
			}
		}
	*/

	metricsLabels[MetricIdStatusCode] = fmt.Sprint(statusCode)
	resp := har.NewResponse(statusCode, statusText, constants.ContentTypeApplicationJson, body, hs)
	return resp, nil
}

const (
	ResponseNotCached = 0
	ResponseCached    = 1
	ResponseCacheHit  = 2
	ResponseCacheMiss = 3
)

func (a *ResponseActivity) handleResponseCache(r *config.Response, resolver *wfcase.ProcessVarResolver, cacheInfo config.CacheInfo) ([]byte, int, error) {

	const semLogContext = string(config.ResponseActivityType) + "::handle-cache"

	if r.Cache.Mode != config.CacheModeGet || r.Cache.Key == "" {
		log.Trace().Str(constants.SemLogCacheKey, cacheInfo.Key).Msg(semLogContext + " not cached response")
		return nil, ResponseNotCached, nil
	}

	var body []byte
	var err error

	respType := ResponseCached
	body, err = a.getCachedResponse(resolver, cacheInfo.BrokerName, cacheInfo.Key)
	if err != nil {
		log.Trace().Str(constants.SemLogCacheKey, cacheInfo.Key).Msg(semLogContext + " cashed response")
		return nil, respType, err
	}

	if body != nil {
		log.Trace().Str(constants.SemLogCacheKey, cacheInfo.Key).Msg(semLogContext + " cache hit response")
		respType = ResponseCacheHit
	} else {
		log.Trace().Str(constants.SemLogCacheKey, cacheInfo.Key).Msg(semLogContext + " cache miss response")
		respType = ResponseCacheMiss
	}

	return body, respType, nil
}

func (a *ResponseActivity) selectResponse(cfg *config.ResponseActivity, wfc *wfcase.WfCase) int {
	const semLogContext = string(config.ResponseActivityType) + "::select-response"
	const semLogResponseId = "response-id"
	for i, r := range cfg.Responses {
		if r.Guard == "" {
			log.Trace().Str(semLogResponseId, r.Id).Msg(semLogContext + " response selected, guard is empty")
			return i
		} else {
			if wfc.EvalExpression(r.Guard) {
				log.Trace().Str(semLogResponseId, r.Id).Str("guard", r.Guard).Msg(semLogContext + " response selected, guard is true")
				return i
			} else {
				log.Trace().Str(semLogResponseId, r.Id).Str("guard", r.Guard).Msg(semLogContext + " response skipped, guard is false")
			}
		}
	}

	return -1
}

func (a *ResponseActivity) computeBody(wfc *wfcase.WfCase, bodyTemplate []byte, resolver *wfcase.ProcessVarResolver) ([]byte, error) {

	s, _, err := varResolver.ResolveVariables(string(bodyTemplate), varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return nil, err
	}

	b, err := wfc.ProcessTemplate(s)
	if err != nil {
		return nil, err
	}

	return b, nil

}

func (a *ResponseActivity) computeHeaders(headers []config.NameValuePair, resolver *wfcase.ProcessVarResolver) ([]har.NameValuePair, error) {

	var resolvedHeaders []har.NameValuePair
	if len(headers) > 0 {
		for _, h := range headers {
			r, _, err := varResolver.ResolveVariables(h.Value, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
			if err != nil {
				return nil, err
			}
			resolvedHeaders = append(resolvedHeaders, har.NameValuePair{Name: h.Name, Value: r})
		}
	}

	return resolvedHeaders, nil

}

func (a *ResponseActivity) getCachedResponse(resolver *wfcase.ProcessVarResolver, redisBrokerName, cacheKey string) ([]byte, error) {

	const semLogContext = string(config.ResponseActivityType) + "::get-cached-response"

	var err error
	cacheKey, _, err = varResolver.ResolveVariables(cacheKey, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return nil, err
	}

	lks, err := linkedservices.GetRedisCacheLinkedService(redisBrokerName)
	if err != nil {
		return nil, err
	}

	v, err := lks.Get(context.Background(), redislks.RedisUseLinkedServiceConfiguredIndex, cacheKey)
	if err != nil {
		log.Error().Err(err).Str("key", cacheKey).Msg(semLogContext + " redis get key error")
		// 2022-05-17. Error is not propagated
		return nil, nil
	}

	if v != nil {
		if b, ok := v.(string); ok {
			log.Trace().Str(constants.SemLogCacheKey, cacheKey).Msg(semLogContext + " cache hit")
			return []byte(b), nil
		}

		log.Error().Err(fmt.Errorf("cache key %s resolves to %T", cacheKey, v)).Send()
	} else {
		log.Warn().Str(constants.SemLogCacheKey, cacheKey).Msg(semLogContext + " cache miss")
	}

	return nil, nil
}

func (a *ResponseActivity) setCachedResponse(resolver *wfcase.ProcessVarResolver, redisBrokerName, cacheKey string, v interface{}) error {

	const semLogContext = string(config.ResponseActivityType) + "::set-cached-response"

	var err error
	cacheKey, _, err = varResolver.ResolveVariables(cacheKey, varResolver.SimpleVariableReference, resolver.ResolveVar, true)
	if err != nil {
		return err
	}

	lks, err := linkedservices.GetRedisCacheLinkedService(redisBrokerName)
	if err != nil {
		return err
	}

	err = lks.Set(context.Background(), redislks.RedisUseLinkedServiceConfiguredIndex, cacheKey, v)
	if err != nil {
		return err
	}

	log.Trace().Str(constants.SemLogCacheKey, cacheKey).Msg(semLogContext + " cache set")
	return nil
}

func (a *ResponseActivity) handleError(err error) error {
	smpErr := smperror.NewExecutableServerError(smperror.WithDescription(err.Error()), smperror.WithErrorAmbit(a.Name()))
	return smpErr
}

func (a *ResponseActivity) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "500",
		MetricIdCached:       "N",
		MetricIdCacheMiss:    "N",
	}

	return metricsLabels
}
