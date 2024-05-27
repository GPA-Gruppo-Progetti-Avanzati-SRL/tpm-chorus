package wfcase

import (
	"encoding/base64"
	"encoding/json"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/rs/zerolog/log"
	"strings"
	"tpm-chorus/orchestration/funcs/cfgdependent"
	"tpm-chorus/orchestration/funcs/simple"
)

func GetFuncMap(wfc *WfCase) map[string]interface{} {
	builtins := make(map[string]interface{})
	builtins["dict"] = func(dict string, elems ...string) string {
		return cfgdependent.Dict(wfc.Dicts, dict, elems...)
	}
	/* Not really needed....
	builtins["var"] = func(varContext string, varReference string) interface{} {
		const semLogContext = "builtins-funcs::var"
		log.Trace().Str("var-ctx", varContext).Str("var-reference", varReference).Msg(semLogContext)
		pvr, err := wfc.GetResolverForEntry(varContext, true, true)
		if err != nil {
			log.Error().Err(err).Str("var-ctx", varContext).Str("var-reference", varReference).Msg(semLogContext)
		}
		v := pvr.ResolveVar("", varReference)
		return v
	}
	*/

	// This funcs is inline and not in the funcs package because it mostly is dependent from the wfcase package and would generate a cyclic dependency...
	builtins["tmpl"] = func(refTmpl string, base64Flag bool, encodeJsonFlag bool) string {
		const semLogContext = "orchestration-funcs::tmpl"

		tmplBody, ok := wfc.Refs.Find(refTmpl)
		if !ok {
			log.Error().Str("ref-template", refTmpl).Msg(semLogContext + " cannot find referenced template")
			return ""
		}

		resolver, err := wfc.GetResolverForEntry("request", true, "", false)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err.Error()
		}

		s, _, err := varResolver.ResolveVariables(string(tmplBody), varResolver.SimpleVariableReference, resolver.ResolveVar, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err.Error()
		}

		b, err := wfc.ProcessTemplate(s)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err.Error()
		}

		resp := string(b)
		if base64Flag {
			resp = base64.StdEncoding.EncodeToString([]byte(resp))
		} else {
			if encodeJsonFlag {
				b, err := json.Marshal(resp)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return err.Error()
				}
				resp = string(b)
			}
		}
		return resp
	}

	builtins["now"] = simple.Now
	builtins["isDate"] = simple.IsDate
	builtins["parseDate"] = simple.ParseDate
	builtins["parseAndFormatDate"] = simple.ParseAndFmtDate
	builtins["dateDiff"] = simple.DateDiff
	builtins["printf"] = simple.Printf
	builtins["amtConv"] = simple.AmtConv
	builtins["amtCmp"] = simple.AmtCmp
	builtins["amtAdd"] = simple.AmtAdd
	builtins["amtDiff"] = simple.AmtDiff
	builtins["padLeft"] = simple.PadLeft
	builtins["left"] = simple.Left
	builtins["right"] = simple.Right
	builtins["len"] = simple.Len
	builtins["isDef"] = simple.IsDefined
	builtins["b64"] = simple.Base64
	builtins["uuid"] = simple.Uuid
	builtins["regexMatch"] = simple.RegexMatch

	return builtins
}

var expressionSmell = []string{
	"dict",
	"isDate",
	"parseDate",
	"parseAndFormatDate",
	"now",
	"printf",
	"amtConv",
	"padLeft",
	"left",
	"right",
	"len",
	"isDef",
	"b64",
	"tmpl",
	">",
	"<",
	"(",
	")",
	"=",
	// "\"",
}

// isExpression In order not to clutter the process vars assignments in simple cases.... try to detect if this is an expression or not.
// didn't parse the thing but try to find if there is any 'reserved' word in there.
// example: 'hello' is not an expression, '"hello"' is an expression which evaluates to 'hello'. This trick is to avoid something like
// value: '"{$.operazione.commissione}"' in the yamls. Someday I'll get to there.... sure...
func isExpression(e string) bool {
	if e == "" {
		return false
	}

	for _, s := range expressionSmell {
		if strings.Contains(e, s) {
			return true
		}
	}

	return false
}