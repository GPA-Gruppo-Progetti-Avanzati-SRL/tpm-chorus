package wfexpressions

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/globals"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/templateutil"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/PaesslerAG/gval"
	"github.com/google/uuid"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/PaesslerAG/jsonpath"
	"github.com/rs/zerolog/log"
)

type EvaluatorOption func(r *Evaluator) error

type Evaluator struct {
	Name string

	vars        ProcessVars
	body        interface{}
	headers     har.NameValuePairs
	queryParams har.NameValuePairs
	params      har.Params

	tempVarsw []string
}

func (pvr *Evaluator) ClearTempVariables() {
	if len(pvr.tempVarsw) > 0 {
		pvr.vars.ClearTemporary(pvr.tempVarsw)
	}
}

func (pvr *Evaluator) VariableLookup(varName string, defaultValue string) (interface{}, bool) {

	var varValue interface{}
	var ok bool
	if len(pvr.vars) > 0 {
		varValue, ok = pvr.vars.Lookup(varName, defaultValue)
	}
	/*
		if !ok && includeTemporary && len(pvr.tempVars) > 0 {
			varValue, ok = pvr.tempVars.Lookup(varName, defaultValue)
		}
	*/

	return varValue, ok
}

/*
	func (pvr *Evaluator) mergeVariables() ProcessVars {
		var resultVars ProcessVars
		if len(pvr.vars) > 0 && len(pvr.tempVars) > 0 {
			resultVars = make(ProcessVars)
			for n, v := range pvr.vars {
				resultVars[n] = v
			}

			for n, v := range pvr.tempVars {
				resultVars[n] = v
			}
		} else {
			if len(pvr.vars) > 0 {
				resultVars = pvr.vars
			} else {
				resultVars = pvr.tempVars
			}
		}

		return resultVars
	}
*/

func (pvr *Evaluator) BodyAsByteArray() ([]byte, error) {
	const semLogContext = "variable-resolver::get-body-as-byte-array"

	var err error
	if pvr.body == nil {
		return nil, nil
	}

	switch bd := pvr.body.(type) {
	case []byte:
		return bd, nil
	case map[string]interface{}:
		var b []byte
		b, err = json.Marshal(bd)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}
		return b, err
	default:
		err = fmt.Errorf("unexpected type for body %T", pvr.body)
		log.Error().Msg(semLogContext)
		return nil, err
	}
}

func (pvr *Evaluator) WithTemporaryProcessVars(tempVars ProcessVars) {
	for n, v := range tempVars {
		pvr.tempVarsw = append(pvr.tempVarsw, n)
		pvr.vars[n] = v
	}
}

func (pvr *Evaluator) WithBody(ct string, aBody []byte, transformationId string) error {
	const semLogContext = "variable-resolver::with-body"
	var err error
	if aBody != nil {
		if strings.HasPrefix(ct, constants.ContentTypeApplicationJson) {
			actualBody := aBody
			if transformationId != "" {
				actualBody, err = kzxform.GetRegistry().Transform(transformationId, aBody)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext + " body transformation failed")
					return err
				}
			}
			v := interface{}(nil)
			err := json.Unmarshal(actualBody, &v)
			if err == nil {
				pvr.body = v
			} else {
				log.Error().Err(err).Msg(semLogContext + " body unmarshal failure")
				log.Error().Str("body", string(actualBody)).Msg(semLogContext + " body unmarshal failure")
				return err
			}
		} else {
			return fmt.Errorf("body content-type is not %s", constants.ContentTypeApplicationJson)
		}
	}

	return nil
}

func WithTemporaryProcessVars(prcVars ProcessVars) EvaluatorOption {
	return func(r *Evaluator) error {
		r.WithTemporaryProcessVars(prcVars)
		return nil
	}
}

func WithProcessVars(prcVars ProcessVars) EvaluatorOption {
	return func(r *Evaluator) error {
		r.vars = prcVars
		return nil
	}
}

func WithBody(ct string, aBody []byte, transformationId string) EvaluatorOption {
	const semLogContext = "variable-resolver::with-body"
	return func(r *Evaluator) error {
		return r.WithBody(ct, aBody, transformationId)
		/*
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
		*/
	}
}

func WithHeaders(h []har.NameValuePair) EvaluatorOption {
	return func(r *Evaluator) error {
		r.headers = h
		return nil
	}
}

func WithQueryParams(h []har.NameValuePair) EvaluatorOption {
	return func(r *Evaluator) error {
		r.queryParams = h
		return nil
	}
}

func WithParams(p []har.Param) EvaluatorOption {
	return func(r *Evaluator) error {
		r.params = p
		return nil
	}
}

func NewEvaluator(aName string, opts ...EvaluatorOption) (*Evaluator, error) {
	pvr := &Evaluator{Name: aName}

	for _, o := range opts {
		err := o(pvr)
		if err != nil {
			return pvr, err
		}
	}

	return pvr, nil
}

var resolverTypePrefix = []string{"$.", "$[", "h:", "p:", "v:", "g:"}

func (pvr *Evaluator) Interpolate(s string) (string, error) {
	val, _, err := varResolver.ResolveVariables(s, varResolver.SimpleVariableReference, pvr.VarResolverFunc, true)
	if err != nil {
		return "", err
	}

	return val, nil
}

func (pvr *Evaluator) InterpolateMany(expr []string) ([]string, error) {
	var resolved []string
	for _, s := range expr {
		val, _, err := varResolver.ResolveVariables(s, varResolver.SimpleVariableReference, pvr.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, val)
	}

	return resolved, nil
}

func (pvr *Evaluator) InterpolateAndEvalToString(s string) (string, error) {
	const semLogContext = "wf-evaluator::interpolate-and-eval-to-string"

	val, err := pvr.InterpolateAndEval(s)
	if err != nil {
		return "", err
	}

	if s, ok := val.(string); ok {
		return s, nil
	}

	log.Warn().Str("expr", s).Str("type", fmt.Sprintf("%T", val)).Msg(semLogContext)
	return fmt.Sprintf("%v", val), nil
}

func (pvr *Evaluator) InterpolateAndEval(s string) (interface{}, error) {
	const semLogContext = "wf-evaluator::interpolate-eval"
	val, _, err := varResolver.ResolveVariables(s, varResolver.SimpleVariableReference, pvr.VarResolverFunc, true)
	if err != nil {
		return "", err
	}

	val, isExpr := pvr.IsExpression(val)
	if isExpr {
		return pvr.Eval(val)
	}

	return val, nil
}

func (pvr *Evaluator) IsExpression(e string) (string, bool) {
	if e == "" {
		return e, false
	}

	if strings.HasPrefix(e, ":") {
		return strings.TrimPrefix(e, ":"), true
	}
	return e, false
}

func (pvr *Evaluator) Eval(s string) (interface{}, error) {
	const semLogContext = "wf-evaluator::eval"

	if s != "" {
		varValue, err := gval.Evaluate(s, pvr.vars)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}
		return varValue, err
	}

	return s, nil
}

func (pvr *Evaluator) EvalToBool(s string) (bool, error) {
	boolVal := true

	if s != "" {
		exprValue, err := gval.Evaluate(s, pvr.vars)
		if err != nil {
			return false, err
		}

		ok := false
		if boolVal, ok = exprValue.(bool); !ok {
			return false, fmt.Errorf("expression %s is not a boolean expression", s)
		}
	}

	return boolVal, nil
}

func (pvr *Evaluator) VarResolverFunc(_, s string) (string, bool) {

	const semLogContext = "process-var-resolver::resolve-var"
	var err error

	doEscape := false
	if strings.HasPrefix(s, "!") {
		doEscape = true
		s = strings.TrimPrefix(s, "!")
	}

	variable, _ := varResolver.ParseVariable(s)
	if variable.Deferred {
		return variable.Raw(), variable.Deferred
	}

	// This is to give the possibility of overriding or extending supported prefixes.
	pfix := string(variable.Prefix)
	if variable.Prefix == varResolver.VariablePrefixNotSpecified {
		pfix, err = pvr.getPrefix(variable.Name)
		if err != nil {
			return "", variable.Deferred
		}
	}

	var varValue interface{}
	var skipVariableOpts bool
	var ok bool
	switch pfix {
	case "$[":
		// Hack because need to change tpm-common...
		// variable.Name = strings.TrimSuffix(variable.Name, "]")
		temp := variable.JsonPathName()
		varValue, err = jsonpath.Get(temp, pvr.body)
		if err == nil {
			ok = true
		}
	case "$.":
		temp := variable.JsonPathName()
		varValue, err = jsonpath.Get(temp, pvr.body)
		if err == nil {
			ok = true
		}
		// log.Trace().Str("path-name", s).Interface("value", v).Msg("evaluation of var")
		/*
			if err == nil {
				s, err = pvr.resolveJsonPathExpr(v)
				if err == nil {
					return pvr.JSONEscape(s, doEscape), false
				}
			}
		*/

		//log.Info().Err(err).Str("path-name", s).Msg(semLogContext + " json-path error")

	case "h:":
		varValue = pvr.headers.GetFirst(variable.Name).Value
		if varValue.(string) != "" {
			ok = true
		}

	case "q:":
		varValue = pvr.queryParams.GetFirst(variable.Name).Value
		if varValue.(string) != "" {
			ok = true
		}
		// return pvr.JSONEscape(s, doEscape), false

	case "p:":
		varValue = pvr.params.GetFirst(variable.Name).Value
		if varValue.(string) != "" {
			ok = true
		}
		// return pvr.JSONEscape(s, doEscape), false

	case "v:":
		vComp := strings.Split(s[2:], ",")
		varValue, ok = pvr.VariableLookup(variable.Name, "")
		if ok {
			if reflect.ValueOf(varValue).Kind() == reflect.Func {
				varValue = pvr.evaluateFunction(varValue, variable.Name, vComp[1:]...)
				skipVariableOpts = true
			} /* else {
				s = fmt.Sprintf("%v", v)
			}
			return pvr.JSONEscape(s, doEscape), false */
		}
	case "g:":
		vComp := strings.Split(s[2:], ",")
		varValue, err = globals.GetGlobalVar("", variable.Name, "")
		if err == nil {
			if reflect.ValueOf(varValue).Kind() == reflect.Func {
				varValue = pvr.evaluateFunction(varValue, variable.Name, vComp[1:]...)
				skipVariableOpts = true
			} /* else {
				s = fmt.Sprintf("%v", v)
			}
			return pvr.JSONEscape(s, doEscape), false */
		} else {
			log.Error().Err(err).Msg(semLogContext)
		}

	default:
		varValue, ok = os.LookupEnv(s)
	}

	if !ok {
		log.Info().Str("var-name", s).Msg(semLogContext + " could not resolve variable!")
	}

	if err != nil {
		if !isJsonPathUnknownKey(err) {
			log.Error().Err(err).Msg(semLogContext)
			return "", variable.Deferred
		}
	}

	if variable.IsTagPresent(varResolver.FormatOptWithTempVar) {
		newName := fmt.Sprintf("%s_%s", "TMP", strings.Replace(uuid.New().String(), "-", "_", -1))
		if pvr.vars == nil {
			pvr.vars = make(map[string]interface{})
		}
		pvr.tempVarsw = append(pvr.tempVarsw, newName)
		pvr.vars[newName] = varValue
		varValue = newName
	}
	s, err = variable.ToString(varValue, doEscape, skipVariableOpts)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	return s, false
}

func (pvr *Evaluator) EvaluateTemplate(tmpl string, funcMap template.FuncMap) ([]byte, error) {
	s, err := pvr.Interpolate(string(tmpl))
	// s, _, err := varResolver.ResolveVariables(string(tmpl), varResolver.SimpleVariableReference, pvr.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}

	ti := []templateutil.Info{
		{
			Name:    "body",
			Content: s,
		},
	}

	pkgTemplate, err := templateutil.Parse(ti, funcMap)
	if err != nil {
		return nil, err
	}

	resp, err := templateutil.Process(pkgTemplate, pvr.vars, false)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func isJsonPathUnknownKey(err error) bool {
	if err != nil {

		s := err.Error()
		switch {
		case strings.HasPrefix(s, "unknown key"):
			return true
		case strings.HasPrefix(s, "unsupported value type <nil> for select"):
			return true
		}
	}

	return false
}

func (pvr *Evaluator) evaluateFunction(v interface{}, funcName string, params ...string) string {
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

/*
	func (pvr *ProcessVarResolver) JSONEscape(s string, doEscape bool) string {
		if doEscape {
			s = util.JSONEscape(s, false)
		}
		return s
	}
*/

/*
func (pvr *Evaluator) resolveJsonPathExpr(v interface{}) (string, error) {

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
*/

func (pvr *Evaluator) getPrefix(s string) (string, error) {

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
	case "g:":
		isValid = true
	case "env":
		isValid = true
	}

	if !isValid {
		return matchedPrefix, fmt.Errorf("found prefix but resover doesn't have data for resolving")
	}

	return matchedPrefix, nil
}
