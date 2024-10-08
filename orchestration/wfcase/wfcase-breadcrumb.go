package wfcase

import "github.com/rs/zerolog/log"

type BreadcrumbStep struct {
	Name        string
	Description string
	Err         error
}

type Breadcrumb []BreadcrumbStep

func (wfc *WfCase) AddBreadcrumb(n string, d string, e error) {
	wfc.Breadcrumb = append(wfc.Breadcrumb, BreadcrumbStep{Name: n, Description: d, Err: e})
}

func (wfc *WfCase) ShowBreadcrumb() {
	for stepNumber, v := range wfc.Breadcrumb {
		log.Trace().Int("at", stepNumber).Str("name", v.Name).Msg("breadcrumb")
		if v.Err != nil {
			log.Trace().Str("with-err", v.Err.Error()).Int("at", stepNumber).Str("name", v.Name).Msg("breadcrumb")
		}
	}
}
