package transformactivity

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type MergeXForm struct {
	Sources []string `yaml:"sources,omitempty"  json:"sources,omitempty" mapstructure:"sources,omitempty"`
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
		var expressionCtx wfcase.ResolverContextReference
		expressionCtx, err = wfc.ResolveExpressionContextName(src)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Trace().Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

		var b []byte
		b, err = wfc.GetBodyByContext(expressionCtx, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		err = json.Unmarshal(b, &m)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
	}

	return json.Marshal(m)
}
