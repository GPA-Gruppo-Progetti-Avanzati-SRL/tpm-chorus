package transformactivity

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type MergeXFormSource struct {
	ActivityName       string `yaml:"activity,omitempty"  json:"activity,omitempty" mapstructure:"activity,omitempty"`
	MergedRootProperty string `yaml:"to-property,omitempty" json:"merged-root-property,omitempty" mapstructure:"merged-root-property,omitempty"`
}

type MergeXForm struct {
	Sources []MergeXFormSource `yaml:"sources,omitempty"  json:"sources,omitempty" mapstructure:"sources,omitempty"`
}

func NewTransformActivityMergeXForm(definition []byte) (MergeXForm, error) {
	xform := MergeXForm{}
	err := yaml.Unmarshal(definition, &xform)
	if err != nil {
		return xform, err
	}

	return xform, nil
}

func (xform MergeXForm) Execute(wfc *wfcase.WfCase, data []byte) ([]byte, error) {
	const semLogContext = "transform-activity-merge-xform::execute"
	var m map[string]interface{}

	err := json.Unmarshal(data, &m)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	for _, src := range xform.Sources {
		var expressionCtx wfcase.HarEntryReference
		expressionCtx, err = wfc.ResolveHarEntryReferenceByName(src.ActivityName)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Trace().Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

		var b []byte
		b, err = wfc.GetBodyInHarEntry(expressionCtx, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		if src.MergedRootProperty == "" {
			err = json.Unmarshal(b, &m)
		} else {
			var temp map[string]interface{}
			err = json.Unmarshal(b, &m)
			if err == nil {
				m[src.MergedRootProperty] = temp
			}
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
	}

	return json.Marshal(m)
}
