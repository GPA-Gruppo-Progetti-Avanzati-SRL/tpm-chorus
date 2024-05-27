package oapiextensions

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"net/http"
)

/*
type SymphonyPathExtension struct {
	OpPath    string `yaml:"op-path" mapstructure:"op-path" json:"op-path"`
	GroupPath string `yaml:"group-path" mapstructure:"group-path" json:"group-path" `
}
*/

type SymphonyOperationExtension struct {
	Path       string `yaml:"path" mapstructure:"path" json:"path"`
	HttpMethod string `yaml:"http-method" mapstructure:"http-method" json:"http-method"`
	Id         string `yaml:"id" mapstructure:"id" json:"id"`
	Comment    string `yaml:"description" mapstructure:"description" json:"description"`
}

func (o SymphonyOperationExtension) IsZero() bool {
	return o.Id == ""
}

func RetrieveSymphonyPathExtensions(doc *openapi3.T) []SymphonyOperationExtension {

	var exts []SymphonyOperationExtension

	for pi, pv := range doc.Paths.Map() {
		oinfos := RetrieveSymphonyPathExtension(pi, pv)
		if len(oinfos) == 0 {
			log.Warn().Str("url", pi).Msg("openapi info without symphony information...skipping")
		} else {
			exts = append(exts, oinfos...)
		}
	}

	return exts
}

func RetrieveSymphonyPathExtension(pi string, pv *openapi3.PathItem) []SymphonyOperationExtension {

	/*
		var pathExt SymphonyPathExtension
		if pv != nil && pv.Extensions != nil {
			v, ok := pv.Extensions["x-symphony"]
			if ok {
				jr, ok := v.(json.RawMessage)
				if ok {
					err := jsoniter.Unmarshal(jr, &pathExt)
					if err != nil {
						log.Error().Err(err).Msg("wrong sid")
					}
				}
			}
		}


		pathExt.OpPath = pi
		if len(pathExt.GroupPath) > 0 && pathExt.GroupPath != "/" {
			if strings.HasPrefix(pi, pathExt.GroupPath) {
				pathExt.OpPath = strings.TrimPrefix(pi, pathExt.GroupPath)
			}
		} else if strings.HasPrefix(pi, "/api/v1") {
			pathExt.GroupPath = "/api/v1"
			pathExt.OpPath = strings.TrimPrefix(pi, pathExt.GroupPath)
		}
	*/

	var oinfo []SymphonyOperationExtension
	httpMethods := []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete}
	for _, m := range httpMethods {
		op := pv.GetOperation(m)
		oi := retrieveSymphonyOperationExtension(pi, m, op)
		if !oi.IsZero() {
			oinfo = append(oinfo, oi)
		} else {
			warnForMissingSymphonyInfo(m, pi, op)
		}
	}

	return oinfo
}

func warnForMissingSymphonyInfo(httpMethod string, path string, op *openapi3.Operation) {
	if op != nil {
		log.Warn().Str("path", path).Str("http-method", httpMethod).Msg("the method has no matching orchestration")
	}
}

func retrieveSymphonyOperationExtension(path string, method string, op *openapi3.Operation) SymphonyOperationExtension {

	if op != nil && op.Extensions != nil {
		v, ok := op.Extensions["x-symphony"]
		if ok {
			m, ok := v.(map[string]interface{})
			if ok {
				var sid SymphonyOperationExtension
				err := mapstructure.Decode(m, &sid)
				if err == nil {
					sid.HttpMethod = method
					sid.Path = path

					sid.Comment = util.StringCoalesce(sid.Comment, op.Summary, op.Description)
					return sid
				}

				log.Error().Err(err).Msg("wrong sid")
			}
		}
	}

	return SymphonyOperationExtension{}
}
