package scriptactivity

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	tengo "github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

const (
	TengoTypePrefixBuiltinFunction  = "builtin-function"
	TengoTypePrefixCompiledFunction = "compiled-function"
	TengoTypePrefixImmutableArray   = "immutable-array"
	TengoTypePrefixImmutableMap     = "immutable-map"
	TengoTypePrefixFreeVar          = "free-var"
	TengoTypePrefixUndefined        = "undefined"
	TengoTypePrefixUserFunction     = "user-function"
)

var TengoMetaTypes = map[string]struct{}{
	TengoTypePrefixBuiltinFunction:  struct{}{},
	TengoTypePrefixCompiledFunction: struct{}{},
	TengoTypePrefixImmutableArray:   struct{}{},
	TengoTypePrefixImmutableMap:     struct{}{},
	TengoTypePrefixFreeVar:          struct{}{},
	TengoTypePrefixUserFunction:     struct{}{},
}

func isValueTypeSupportedType(t string) bool {
	ndx := strings.Index(t, ":")
	if ndx >= 0 {
		t = t[:ndx]
	}

	if _, ok := TengoMetaTypes[t]; ok {
		return false
	}

	return true
}

type ScriptActivity struct {
	executable.Activity
	definition config.ScriptActivityDefinition
}

func NewScriptActivity(item config.Configurable, refs config.DataReferences) (*ScriptActivity, error) {
	var err error

	ea := &ScriptActivity{}
	ea.Cfg = item
	ea.Refs = refs

	eaCfg, ok := item.(*config.ScriptActivity)
	if !ok {
		err := fmt.Errorf("this is weird %T is not %s config type", item, config.ScriptActivityType)
		return nil, err
	}

	ea.definition, err = config.UnmarshalScriptActivityDefinition(eaCfg.Definition, refs)
	if err != nil {
		return nil, err
	}

	return ea, nil
}

func (a *ScriptActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.ScriptActivityType) + "::execute"
	var err error

	if !a.IsEnabled(wfc) {
		log.Info().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.ScriptActivityType)).Msg("activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Str("msg", a.definition.Script).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.ScriptActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.ScriptActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	_, _, err = a.MetricsGroup()
	if err != nil {
		log.Error().Err(err).Interface("metrics-config", a.Cfg.MetricsConfig()).Msg(semLogContext + " cannot found metrics group")
		return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
	}

	beginOf := time.Now()
	metricsLabels := a.MetricsLabels()
	defer func() {
		a.SetMetrics(beginOf, metricsLabels)
	}()

	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(a.Cfg.ExpressionContextNameStringReference())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}

	log.Trace().Str(constants.SemLogActivity, a.Name()).Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

	if len(tcfg.ProcessVars) > 0 {
		err := wfc.SetVars(expressionCtx, tcfg.ProcessVars, "", false)
		if err != nil {
			wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
			return smperror.NewExecutableServerError(smperror.WithErrorAmbit(a.Name()), smperror.WithErrorMessage(err.Error()))
		}
	}

	script, bdy, err := a.computeScript(wfc)
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return err
	}

	req, _ := a.newRequestDefinition([]byte(bdy))
	_ = wfc.SetHarEntryRequest(a.Name(), req, config.PersonallyIdentifiableInformation{})

	compiled, err := script.RunContext(context.Background())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		resp := har.NewResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), constants.ContentTypeTextPlain, []byte(err.Error()), nil)
		_ = wfc.SetHarEntryResponse(a.Name(), resp, config.PersonallyIdentifiableInformation{})
		return err
	}

	scriptTengoOutVars := compiled.GetAll()

	var scriptOutVars map[string]interface{}
	if len(scriptTengoOutVars) > 0 {
		scriptOutVars = make(map[string]interface{})
		for _, v := range scriptTengoOutVars {
			if isValueTypeSupportedType(v.ValueType()) {
				scriptOutVars[v.Name()] = v.Value()
			}
		}
	}

	resp, err := a.newResponseDefinition(scriptOutVars)
	_ = wfc.SetHarEntryResponse(a.Name(), resp, config.PersonallyIdentifiableInformation{})

	remappedStatusCode, err := a.processResponseActions(http.StatusOK, a.Name(), a.Name(), wfc, a.definition.OnResponseActions, scriptOutVars)
	if remappedStatusCode > 0 {
		metricsLabels[MetricIdStatusCode] = fmt.Sprint(remappedStatusCode)
	}
	if err != nil {
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		return err
	}

	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)

	return nil
}

func (a *ScriptActivity) newResponseDefinition(scriptVars wfexpressions.ProcessVars) (*har.Response, error) {
	b, err := json.Marshal(scriptVars)
	if err != nil {
		return nil, nil
	}

	resp := har.NewResponse(http.StatusOK, http.StatusText(http.StatusOK), constants.ContentTypeApplicationJson, []byte(b), nil)
	return resp, nil
}

func (a *ScriptActivity) newRequestDefinition(body []byte) (*har.Request, error) {
	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(0)
	ub.WithScheme("activity")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s", string(config.ScriptActivityType), a.Name()))

	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))

	opts = append(opts, har.WithBody(body))

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

func (a *ScriptActivity) computeScript(wfc *wfcase.WfCase) (*tengo.Script, string, error) {
	const semLogContext = "script-activity::compute-body"

	evaluator, err := a.GetEvaluator(wfc)
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return nil, "", err
	}

	text, err := evaluator.EvaluateTemplate(string(a.definition.ScriptText), wfc.TemplateFunctions())
	if err != nil {
		log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		return nil, "", err
	}

	var sb strings.Builder
	sb.WriteString("\n================= Script Text: \n")
	sb.WriteString(string(text))

	script := tengo.NewScript(text)
	if len(a.definition.StdLibModules) > 0 {
		script.SetImports(stdlib.GetModuleMap(a.definition.StdLibModules...))
	}

	var paramsMap map[string]interface{}
	for _, p := range a.definition.Params {
		paramVal, err := evaluator.InterpolateAndEval(p.Value)
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
			return nil, "", err
		}
		_ = script.Add(p.Name, paramVal)
		if paramsMap == nil {
			paramsMap = make(map[string]interface{})
		}
		paramsMap[p.Name] = paramVal
	}

	if len(paramsMap) > 0 {
		b, err := json.Marshal(paramsMap)
		if err != nil {
			return nil, "", err
		}

		sb.WriteString("\n================= Script Params: \n")
		sb.WriteString(string(b))
	}

	return script, sb.String(), nil
}

func (a *ScriptActivity) processResponseActions(
	st int,
	ambitName, stepName string,
	wfc *wfcase.WfCase,
	actions config.OnResponseActions,
	scriptVars wfexpressions.ProcessVars,
) (int, error) {

	const semLogContext = "script-activity::process-response-action-by-status-code"
	actNdx := actions.FindByStatusCode(st)
	if actNdx < 0 {
		return -1, nil
	}

	act := actions[actNdx]

	evaluator, err := wfc.GetEvaluatorByHarEntryReference(wfcase.HarEntryReference{Name: a.Name(), UseResponse: true}, true, "", true)
	if err != nil {
		log.Error().Err(err).Str("ctx", stepName).Msg(semLogContext)
		return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambitName), smperror.WithStep(stepName), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
	}

	if len(act.ProcessVars) > 0 {
		evaluator.WithTemporaryProcessVars(scriptVars)
		for _, v := range act.ProcessVars {
			boolGuard := true
			if v.Guard != "" {
				boolGuard, err = evaluator.EvalToBool(v.Guard)
			}

			if boolGuard && err == nil {
				val, err := evaluator.InterpolateAndEval(v.Value)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambitName), smperror.WithStep(stepName), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
				}

				err = wfc.Vars.Set(v.Name, val, v.GlobalScope, v.Ttl)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambitName), smperror.WithStep(stepName), smperror.WithCode("500"), smperror.WithErrorMessage("error selecting transformation"), smperror.WithDescription(err.Error()))
				}
			}
		}

		evaluator.ClearTempVariables()
	}

	if ndx := a.ChooseError(wfc, act.Errors); ndx >= 0 {

		e := act.Errors[ndx]
		ambit := e.Ambit
		if ambit == "" {
			ambit = ambitName
		}

		step := e.Step
		if step == "" {
			step = stepName
		}

		statusCode := st
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}

		m, err := evaluator.InterpolateMany([]string{e.Code, e.Message, e.Description, step})
		if err != nil {
			log.Error().Err(err).Msgf("error resolving values %s, %s and %s", e.Code, e.Message, e.Description)
			return 500, smperror.NewExecutableError(smperror.WithErrorStatusCode(500), smperror.WithErrorAmbit(ambit), smperror.WithStep(step), smperror.WithCode(e.Code), smperror.WithErrorMessage(e.Message), smperror.WithDescription(err.Error()))
		}
		return statusCode, smperror.NewExecutableError(smperror.WithErrorStatusCode(statusCode), smperror.WithErrorAmbit(ambit), smperror.WithStep(m[3]), smperror.WithCode(m[0]), smperror.WithErrorMessage(m[1]), smperror.WithDescription(m[2]))
	}

	return 0, nil
}

const (
	MetricIdActivityType           = "type"
	MetricIdActivityName           = "name"
	MetricIdEndpointDefinitionPath = "endpoint"
	MetricIdEndpointId             = "endpoint-id"
	MetricIdEndpointName           = "endpoint-name"
	MetricIdStatusCode             = "status-code"
	MetricIdMethod                 = "http-method"
	MetricIdHttpStatusCode         = "http-status-code"
)

func (a *ScriptActivity) MetricsLabels() prometheus.Labels {
	cfg := a.Cfg.(*config.ScriptActivity)
	metricsLabels := prometheus.Labels{
		MetricIdActivityType: string(cfg.Type()),
		MetricIdActivityName: a.Name(),
		MetricIdStatusCode:   "-1",
	}

	return metricsLabels
}
