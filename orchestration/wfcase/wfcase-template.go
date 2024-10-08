package wfcase

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/templateutil"
	"text/template"
)

func (wfc *WfCase) TemplateFunctions() template.FuncMap {

	fMap := template.FuncMap(GetFuncMap(wfc))
	/* {
		"dict": func(dictName string, elems ...string) string {
			s, err := wfc.Dicts.Map(dictName, elems...)
			if err != nil {
				log.Error().Err(err).Send()
			}
			return s
		},
		"now": func(format string) string {
			return time.Now().Format(format)
		},
	}*/

	return fMap
}

func (wfc *WfCase) ProcessTemplate(tmpl string) ([]byte, error) {

	ti := []templateutil.Info{
		{
			Name:    "body",
			Content: tmpl,
		},
	}

	pkgTemplate, err := templateutil.Parse(ti, wfc.TemplateFunctions())
	if err != nil {
		return nil, err
	}

	resp, err := templateutil.Process(pkgTemplate, wfc.Vars, false)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
