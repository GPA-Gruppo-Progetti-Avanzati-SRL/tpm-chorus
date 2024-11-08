package echoactivity

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks/gocachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelksregistry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/smperror"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/rs/zerolog/log"
	"net/http"
	"reflect"
)

type EchoActivity struct {
	executable.Activity
	definition config.EchoActivityDefinition
}

func NewEchoActivity(item config.Configurable, refs config.DataReferences) (*EchoActivity, error) {
	var err error

	ea := &EchoActivity{}
	ea.Cfg = item
	ea.Refs = refs

	eaCfg, ok := item.(*config.EchoActivity)
	if !ok {
		err := fmt.Errorf("this is weird %T is not %s config type", item, config.EchoActivityType)
		return nil, err
	}

	ea.definition, err = config.UnmarshalEchoActivityDefinition(eaCfg.Definition, refs)
	if err != nil {
		return nil, err
	}

	if ea.definition.IsZero() {
		ea.definition.Message = eaCfg.Message
	}
	return ea, nil
}

func (a *EchoActivity) Execute(wfc *wfcase.WfCase) error {
	const semLogContext = string(config.EchoActivityType) + "::execute"
	var err error
	if !a.IsEnabled(wfc) {
		log.Info().Str(constants.SemLogActivity, a.Name()).Str("type", string(config.EchoActivityType)).Msg("activity not enabled")
		return nil
	}

	log.Info().Str(constants.SemLogActivity, a.Name()).Msg(semLogContext + " start")
	defer log.Info().Str(constants.SemLogActivity, a.Name()).Str("msg", a.definition.Message).Msg(semLogContext + " end")

	tcfg, ok := a.Cfg.(*config.EchoActivity)
	if !ok {
		err = fmt.Errorf("this is weird %T is not %s config type", a.Cfg, config.EchoActivityType)
		wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), err)
		log.Error().Err(err).Msg(semLogContext)
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

	if a.definition.IncludeInHar {
		req, _ := a.newRequestDefinition(wfc)
		_ = wfc.SetHarEntryRequest(a.Name(), req, config.PersonallyIdentifiableInformation{})

		ct, b, err := a.computeBody(wfc)
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
			return err
		}
		resp := har.NewResponse(http.StatusOK, http.StatusText(http.StatusOK), ct, []byte(b), nil)
		_ = wfc.SetHarEntryResponse(a.Name(), resp, config.PersonallyIdentifiableInformation{})
	}

	wfc.AddBreadcrumb(a.Name(), a.Cfg.Description(), nil)
	return nil
}

/*func (a *EchoActivity) harEntry(wfc *wfcase.WfCase) (*har.Entry, error) {
	now := time.Now()

	req, err := a.newRequestDefinition(wfc)
	if err != nil {
		return nil, err
	}

	ct, b, err := a.computeBody(wfc)
	if err != nil {
		return nil, err
	}
	resp := har.NewResponse(http.StatusOK, http.StatusText(http.StatusOK), ct, b, nil)

	e := &har.Entry{
		StartedDateTime: now.Format(time.RFC3339Nano),
		StartDateTimeTm: now,
		Request:         req,
		Response:        resp,
	}

	return e, nil
}*/

func (a *EchoActivity) newRequestDefinition(wfc *wfcase.WfCase) (*har.Request, error) {
	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(0)
	ub.WithScheme("activity")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s", string(config.EchoActivityType), a.Name()))

	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))

	b, err := json.Marshal(a.definition)
	if err != nil {
		return nil, err
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

func (a *EchoActivity) computeBody(wfc *wfcase.WfCase) (string, []byte, error) {
	const semLogContext = "echo-activity::compute-body"

	body := make(map[string]interface{})
	body["message"] = a.definition.Message
	if a.definition.WithVars {
		caseVariables := make(map[string]interface{})
		for n, v := range wfc.Vars {
			if reflect.ValueOf(v).Kind() != reflect.Func {
				caseVariables[n] = v
			}
		}
		body["process-vars"] = caseVariables
	}

	if a.definition.WithGoCache != "" {
		items, err := cachelksregistry.GetItems4Cache(gocachelks.GoCacheLinkedServiceType, a.definition.WithGoCache)
		if err != nil {
			log.Error().Err(err).Str(constants.SemLogActivity, a.Name()).Msg(semLogContext)
		} else {
			if len(items) > 0 {
				body[a.definition.WithGoCache] = items
			}
		}
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", nil, err
	}

	return constants.ContentTypeApplicationJson, b, nil
}
