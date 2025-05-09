package mongoactivity

import (
	"context"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cacheoperation"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/mongolks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"time"
)

type MongoActivity struct {
	executable.Activity
	definition config.MongoActivityDefinition
}

func NewMongoActivity(item config.Configurable, refs config.DataReferences) (*MongoActivity, error) {
	const semLogContext = "mongo-activity::new"
	var err error

	ma := &MongoActivity{}
	ma.Cfg = item
	ma.Refs = refs

	maCfg := item.(*config.MongoActivity)
	ma.definition, err = config.UnmarshalMongoActivityDefinition(maCfg.OpType, maCfg.Definition, refs)
	if err != nil {
		return nil, err
	}

	return ma, nil
}

func (a *MongoActivity) Execute(wfc *wfcase.WfCase) error {

	const semLogContext = string(config.MongoActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Trace().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.MongoActivityType)).Msg(semLogContext + " activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.MongoActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.MongoActivityType)
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

	if len(tcfg.ProcessVars) > 0 {
		expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
			return err
		}
		log.Trace().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")

		err = wfc.SetVars(expressionCtx, tcfg.ProcessVars, "", false)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func() { a.SetMetrics(beginOf, metricsLabels) }()

	resolver, err := a.GetEvaluator(wfc)
	if err != nil {
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	var harResponse *har.Response
	var cacheCfg config.CacheConfig
	var cacheEnabled bool
	cacheEnabled, err = a.definition.CacheConfig.Enabled()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}
	if cacheEnabled {
		cacheCfg, err = a.resolveCacheConfig(wfc, resolver, a.definition.CacheConfig, a.Refs)
		if err != nil {
			// The get of the cache triggers an error only.
			log.Error().Err(err).Msg(semLogContext)
		} else {
			harResponse, err = a.resolveResponseFromCache(wfc, cacheCfg)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				if harResponse == nil {
					return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
				}
			}
		}
	}

	var mongoError int
	if harResponse == nil || harResponse.Status != http.StatusOK {
		statementConfig, err := a.definition.LoadStatementConfig(a.Refs)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		statementConfig, err = a.resolveStatementParts(wfc, statementConfig)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		op, err := jsonops.NewOperation(tcfg.OpType, statementConfig)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		req, err := a.newRequestDefinition(wfc, op) // TODO calcolare lo statement
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			metricsLabels[MetricIdStatusCode] = "500"
			// See defer a.SetMetrics(beginOf, metricsLabels)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithErrorMessage(err.Error()))
		}

		_ = wfc.SetHarEntryRequest(a.Name(), req, tcfg.PII)

		harResponse, mongoError, err = a.Invoke(wfc, op)
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		}

		/*
			if err != nil {
				wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
				metricsLabels[MetricIdStatusCode] = fmt.Sprint(mongoError)
				// See defer a.SetMetrics(beginOf, metricsLabels)
				return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithStep(a.Name()), smperror.WithCode("MONGO"), smperror.WithErrorMessage(err.Error()))
			}
		*/

		if harResponse != nil {
			_ = wfc.SetHarEntryResponse(a.Name(), harResponse, tcfg.PII)
			metricsLabels[MetricIdStatusCode] = fmt.Sprint(mongoError)

			if cacheEnabled && harResponse.Status == http.StatusOK {
				err = a.saveResponseToCache(cacheCfg, harResponse.Content.Data)
				// err = cacheoperation.Set(cacheCfg.LinkedServiceRef, cacheCfg.Key, harResponse.Content.Data, cachelks.WithNamespace(cacheCfg.Namespace))
				if err != nil {
					// The set of the cache triggers an error only.
					log.Error().Err(err).Msg(semLogContext)
				}
			}
		}
	}

	statusToRemap := harResponse.Status
	if harResponse.Status != http.StatusOK && mongoError != 0 {
		statusToRemap = mongoError
	}
	remappedStatusCode, err := a.ProcessResponseActionByStatusCode(
		statusToRemap, a.Name(), a.Name(), wfc, nil, wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}, a.definition.OnResponseActions, false)
	if remappedStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
	}
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		// see defer_ = a.SetMetrics(beginOf, metricsLabels)
		return err
	}

	/*
		actNdx := a.definition.OnResponseActions.FindByStatusCode(harResponse.Status)
		if actNdx >= 0 {
			remappedStatusCode, err := a.processResponseAction(wfc, a.Name(), actNdx, harResponse)
			if remappedStatusCode != 0 {
				metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
			}
			if err != nil {
				wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
				a.SetMetrics(beginOf, metricsLabels)
				return err
			}
		}
	*/

	// See defer _ = a.SetMetrics(beginOf, metricsLabels)
	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)

	return err
}

func (a *MongoActivity) resolveStatementParts(wfc *wfcase.WfCase, m map[jsonops.MongoJsonOperationStatementPart][]byte) (map[jsonops.MongoJsonOperationStatementPart][]byte, error) {

	/*
		expressionCtx, err := wfc.ResolveExpressionContextName(a.Cfg.ExpressionContextNameStringReference())
		if err != nil {
			return nil, err
		}

		resolver, err := wfc.GetResolverByContext(expressionCtx, true, "", false)
	*/
	resolver, err := a.GetEvaluator(wfc)
	if err != nil {
		return nil, err
	}

	newMap := map[jsonops.MongoJsonOperationStatementPart][]byte{}
	for n, b := range m {
		s, _, err := varResolver.ResolveVariables(string(b), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}

		b1, err := wfc.ProcessTemplate(s)
		if err != nil {
			return nil, err
		}

		newMap[n] = b1
	}

	return newMap, nil
}

func (a *MongoActivity) Invoke(wfc *wfcase.WfCase, op jsonops.Operation) (*har.Response, int, error) {

	const semLogContext = "mongo-activity::invoke"
	lks, err := mongolks.GetLinkedService(context.Background(), a.definition.LksName)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, -1, err
	}

	sc, resp, err := op.Execute(lks, a.definition.CollectionId)

	var r *har.Response
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		err = util.NewError(strconv.Itoa(sc.StatusCode), err)
		r = har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "text/plain", []byte(err.Error()), nil)
		return r, sc.StatusCode, err
	}

	on200ActionNdx := a.definition.OnResponseActions.FindByStatusCode(http.StatusOK)
	if on200ActionNdx >= 0 && len(a.definition.OnResponseActions[on200ActionNdx].Properties) > 0 {
		onResponseProperties := a.definition.OnResponseActions[on200ActionNdx].Properties
		if varName, ok := onResponseProperties[config.MongoOperationResultMatchedCountPropertyVarName]; ok {
			wfc.Vars[varName] = sc.MatchedCount
		}

		if varName, ok := onResponseProperties[config.MongoOperationResultUpsertedCountPropertyVarNName]; ok {
			wfc.Vars[varName] = sc.UpsertedCount
		}

		if varName, ok := onResponseProperties[config.MongoOperationResultModifiedCountPropertyVarName]; ok {
			wfc.Vars[varName] = sc.ModifiedCount
		}

		if varName, ok := onResponseProperties[config.MongoOperationResultDeletedCountPropertyVarName]; ok {
			wfc.Vars[varName] = sc.DeletedCount
		}

		if varName, ok := onResponseProperties[config.MongoOperationResultObjectIDPropertyVarName]; ok {
			wfc.Vars[varName] = sc.ObjectID
		}
	}

	r = &har.Response{
		Status:      sc.StatusCode,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(sc.StatusCode),
		HeadersSize: -1,
		BodySize:    int64(len(resp)),
		Cookies:     []har.Cookie{},
		Headers:     []har.NameValuePair{},
		Content: &har.Content{
			MimeType: constants.ContentTypeApplicationJson,
			Size:     int64(len(resp)),
			Data:     resp,
		},
	}

	return r, 0, nil
}

func (a *MongoActivity) newRequestDefinition(wfc *wfcase.WfCase, op jsonops.Operation) (*har.Request, error) {

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(27017)
	ub.WithScheme("mongodb")

	ub.WithHostname("localhost")
	ub.WithPath("/" + string(a.definition.OpType))
	ub.WithPath(fmt.Sprintf("/%s/%s/%s", string(config.MongoActivityType), string(a.definition.OpType), a.Name()))
	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))
	opts = append(opts, har.WithBody([]byte(op.ToString())))

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

func (a *MongoActivity) MetricsLabels() prometheus.Labels {

	cfg := a.Cfg.(*config.MongoActivity)
	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(a.Cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdOpType:       string(cfg.OpType),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}

/*
func (a *MongoActivity) processResponseAction(wfc *wfcase.WfCase, activityName string, actionIndex int, resp *har.Response) (int, error)  {
	const semLogContext = "mongo-activity::processResponseAction"

	act := a.definition.OnResponseActions[actionIndex]

	transformId, err := chooseTransformation(wfc, act.Transforms)
	if err != nil {
		log.Error().Err(err).Str("request-id", wfc.GetRequestId()).Msg(semLogContext + " - error in selecting transformation")
		return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(a.Name()), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
	}

	contextReference := wfcase.ResolverContextReference{Name: a.Name(), UseResponse: true}

	if len(act.ProcessVars) > 0 {
		err := wfc.SetVars(contextReference, act.ProcessVars, transformId, false)
		if err != nil {
			log.Error().Err(err).Str("ctx", a.Name()).Str("request-id", wfc.GetRequestId()).Msg(semLogContext + " -  error in setting variables")
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(activityName), smperror.WithStep(a.Name()), smperror.WithCode("500"), smperror.WithErrorMessage("error processing response body"), smperror.WithDescription(err.Error()))
		}
	}

	if ndx := chooseError(wfc, act.Errors); ndx >= 0 {

		e := act.Errors[ndx]
		ambit := e.Ambit
		if ambit == "" {
			ambit = activityName
		}

		step := e.Step
		if step == "" {
			step = a.Name()
		}
		if step == "" {
			step = a.Name()
		}

		statusCode := int(resp.Status)
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}

		m, err := wfc.ResolveStrings(contextReference, []string{e.Code, e.Message, e.Description, step}, "", false)
		if err != nil {
			log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
		}
		return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]))
	}

	return 0, nil
}

func chooseTransformation(wfc *wfcase.WfCase, trs []transform.TransformReference) (string, error) {
	for _, t := range trs {

		b := true
		if t.Guard != "" {
			b = wfc.EvalExpression(t.Guard)
		}

		if b {
			return t.Id, nil
		}
	}

	return "", nil
}

func chooseError(wfc *wfcase.WfCase, errors []config.ErrorInfo) int {
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


func (a *MongoActivity) findResponseAction(statusCode int) int {

	matchedAction := -1
	defaultAction := -1
	for ndx, act := range a.definition.OnResponseActions {
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
} */

func (a *MongoActivity) resolveCacheConfig(wfc *wfcase.WfCase, resolver *wfexpressions.Evaluator, cacheConfig config.CacheConfig, refs config.DataReferences) (config.CacheConfig, error) {
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

func (a *MongoActivity) resolveResponseFromCache(wfc *wfcase.WfCase, cacheConfig config.CacheConfig) (*har.Response, error) {
	cacheHarEntry, err := cacheoperation.Get(
		cacheConfig.LinkedServiceRef,
		a.Name()+";cache=true",
		cacheConfig.Key,
		constants.ContentTypeApplicationJson,
		cachelks.WithNamespace(cacheConfig.Namespace), cachelks.WithHarPath(fmt.Sprintf("/%s/%s/%s;cache=true", string(config.MongoActivityType), string(a.definition.OpType), a.Name())))
	if err != nil {
		return nil, err
	}

	// the id takes the activity name in case ok because no other entry will be present. In case of cache miss ad additional entry will be there
	// together with the un-cached invokation
	entryId := a.Name()
	if cacheHarEntry.Response.Status != http.StatusOK {
		entryId = a.Name() + ";cache=true"
	}

	_ = wfc.SetHarEntry(entryId, cacheHarEntry)
	return cacheHarEntry.Response, nil
}

func (a *MongoActivity) saveResponseToCache(cacheConfig config.CacheConfig, data []byte) error {
	err := cacheoperation.Set(cacheConfig.LinkedServiceRef, cacheConfig.Key, data, cachelks.WithNamespace(cacheConfig.Namespace), cachelks.WithTTTL(cacheConfig.Ttl))
	if err != nil {
		return err
	}

	return nil
}
