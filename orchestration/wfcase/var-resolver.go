package wfcase

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"os"
	"reflect"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/rs/zerolog/log"
)

type VarResolverOption func(r *ProcessVarResolver) error

type ProcessVarResolver struct {
	vars    ProcessVars
	body    interface{}
	headers har.NameValuePairs
	params  har.Params
}

func WithProcessVars(prcVars ProcessVars) VarResolverOption {
	return func(r *ProcessVarResolver) error {
		r.vars = prcVars
		return nil
	}
}

func WithBody(ct string, aBody []byte, transformationId string) VarResolverOption {
	const semLogContext = "variable-resolver::with-body"
	return func(r *ProcessVarResolver) error {
		var err error
		if aBody != nil {
			if strings.HasPrefix(ct, constants.ContentTypeApplicationJson) {
				actualBody := aBody
				if transformationId != "" {
					actualBody, err = transform.GetRegistry().Transform(transformationId, aBody)
					if err != nil {
						log.Error().Err(err).Msg(semLogContext + " body transformation failed")
						return err
					}
				}
				v := interface{}(nil)
				err := json.Unmarshal(actualBody, &v)
				if err == nil {
					r.body = v
				} else {
					return err
				}
			} else {
				return fmt.Errorf("body content-type is not %s", constants.ContentTypeApplicationJson)
			}
		}

		return nil
	}
}

func WithHeaders(h []har.NameValuePair) VarResolverOption {
	return func(r *ProcessVarResolver) error {
		r.headers = h
		return nil
	}
}

func WithParams(p []har.Param) VarResolverOption {
	return func(r *ProcessVarResolver) error {
		r.params = p
		return nil
	}
}

func NewProcessVarResolver(opts ...VarResolverOption) (*ProcessVarResolver, error) {
	pvr := &ProcessVarResolver{}

	for _, o := range opts {
		err := o(pvr)
		if err != nil {
			return pvr, err
		}
	}

	return pvr, nil
}

var resolverTypePrefix = []string{"$.", "$[", "h:", "p:", "v:"}

func (pvr *ProcessVarResolver) ResolveVar(_, s string) (string, bool) {

	const semLogContext = "process-var-resolver::resolve-var"
	doEscape := false
	if strings.HasPrefix(s, "!") {
		doEscape = true
		s = strings.TrimPrefix(s, "!")
	}

	pfix, err := pvr.getPrefix(s)
	if err != nil {
		return "", false
	}

	switch pfix {
	case "$[":
		fallthrough
	case "$.":
		var v interface{}
		v, err = jsonpath.Get(s, pvr.body)
		// log.Trace().Str("path-name", s).Interface("value", v).Msg("evaluation of var")
		if err == nil {
			s, err = pvr.resolveJsonPathExpr(v)
			if err == nil {
				return pvr.JSONEscape(s, doEscape), false
			}
		}

		//log.Info().Err(err).Str("path-name", s).Msg(semLogContext + " json-path error")

	case "h:":
		s = pvr.headers.GetFirst(s[2:]).Value
		return pvr.JSONEscape(s, doEscape), false

	case "p:":
		s = pvr.params.GetFirst(s[2:]).Value
		return pvr.JSONEscape(s, doEscape), false

	case "v:":
		vComp := strings.Split(s[2:], ",")
		v, ok := pvr.vars.Get(vComp[0])
		if ok {
			if reflect.ValueOf(v).Kind() == reflect.Func {
				s = pvr.resolveFunctionVar(v, vComp[0], vComp[1:]...)
			} else {
				s = fmt.Sprintf("%v", v)
			}
			return pvr.JSONEscape(s, doEscape), false
		}

	default:
		v, ok := os.LookupEnv(s)
		if ok {
			return pvr.JSONEscape(v, doEscape), false
		}
	}

	log.Info().Str("var-name", s).Msg("could not resolve variable")
	return "", false
}

func (pvr *ProcessVarResolver) resolveFunctionVar(v interface{}, funcName string, params ...string) string {
	const semLogContext = "process-var-resolver::resolve-func-var"
	log.Trace().Interface("kind", reflect.ValueOf(v).Kind()).Msg(semLogContext)

	var s string

	if f, ok := v.(func() string); ok {
		s = f()
		return s
	}

	if f, ok := v.(func(string) string); ok {
		var p string
		if len(params) > 0 {
			p = params[0]
		}
		s = f(p)
		return s
	}

	if f, ok := v.(func(string, string) string); ok {
		var p1, p2 string
		switch len(params) {
		case 0:
		case 1:
			p1 = params[0]
		default:
			p1 = params[0]
			p2 = params[0]
		}

		s = f(p1, p2)
		return s
	}

	if f, ok := v.(func(s string, params ...string) string); ok {
		var p string
		var ps []string
		switch len(params) {
		case 0:
		case 1:
			p = params[0]
		default:
			p = params[0]
			ps = params[1:]
		}

		s = f(p, ps...)
		return s
	}

	log.Warn().Str("func-name", funcName).Msg(semLogContext + " function signature not supported")
	return s
}

func (pvr *ProcessVarResolver) JSONEscape(s string, doEscape bool) string {
	if doEscape {
		s = util.JSONEscape(s)
	}
	return s
}

func (pvr *ProcessVarResolver) resolveJsonPathExpr(v interface{}) (string, error) {

	var s string
	var err error
	if v != nil {
		var b []byte
		switch v.(type) {
		case float64, float32:
			s = fmt.Sprintf("%f", v)
		case map[string]interface{}:
			b, err = json.Marshal(v)
			if err == nil {
				s = string(b)
			}
		case []interface{}:
			b, err = json.Marshal(v)
			if err == nil {
				s = string(b)
			}
		default:
			s = fmt.Sprintf("%v", v)
		}
	}

	return s, err
}

func (pvr *ProcessVarResolver) getPrefix(s string) (string, error) {

	matchedPrefix := "env"

	for _, pfix := range resolverTypePrefix {
		if strings.HasPrefix(s, pfix) {
			matchedPrefix = pfix
			break
		}
	}

	isValid := false
	switch matchedPrefix {
	case "$[":
		fallthrough
	case "$.":
		if pvr.body != nil {
			isValid = true
		}

	case "h:":
		if pvr.headers != nil {
			isValid = true
		}

	case "p:":
		if pvr.params != nil {
			isValid = true
		}
	case "v:":
		if pvr.vars != nil {
			isValid = true
		}
	case "env":
		isValid = true
	}

	if !isValid {
		return matchedPrefix, fmt.Errorf("found prefix but resover doesn't have data for resolving")
	}

	return matchedPrefix, nil
}
