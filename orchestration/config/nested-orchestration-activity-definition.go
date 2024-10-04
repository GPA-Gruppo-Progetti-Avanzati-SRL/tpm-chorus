package config

import (
	"errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type NestedOrchestrationActivityDefinition struct {
	OrchestrationId   string            `yaml:"orchestration-id,omitempty" json:"orchestration-id,omitempty" mapstructure:"orchestration-id,omitempty"`
	OnResponseActions OnResponseActions `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func UnmarshalNestedOrchestrationActivityDefinition(def string, refs DataReferences) (NestedOrchestrationActivityDefinition, error) {
	const semLogContext = "nested-orchestration-activity-definition::unmarshal"

	var err error
	maDef := NestedOrchestrationActivityDefinition{}

	if def != "" {
		data, ok := refs.Find(def)
		if len(data) == 0 || !ok {
			err = errors.New("cannot find activity definition")
			log.Error().Err(err).Msg(semLogContext)
			return maDef, err
		}

		err = yaml.Unmarshal(data, &maDef)
		if err != nil {
			return maDef, err
		}
	}

	return maDef, nil
}
