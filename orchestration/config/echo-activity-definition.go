package config

import (
	"errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type EchoActivityDefinition struct {
	Message      string `yaml:"message,omitempty" json:"message,omitempty" mapstructure:"message,omitempty"`
	IncludeInHar bool   `yaml:"in-har,omitempty" json:"in-har,omitempty" mapstructure:"in-har,omitempty"`
	ShowVars     bool   `yaml:"display-vars,omitempty" json:"display-vars,omitempty" mapstructure:"display-vars,omitempty"`
}

func (def *EchoActivityDefinition) IsZero() bool {
	return def.Message == "" && def.IncludeInHar == false
}

func UnmarshalEchoActivityDefinition(def string, refs DataReferences) (EchoActivityDefinition, error) {
	const semLogContext = "echo-activity-definition::unmarshal"

	var err error
	maDef := EchoActivityDefinition{}

	if def != "" {
		data, ok := refs.Find(def)
		if len(data) == 0 || !ok {
			err = errors.New("cannot find mongo activity definition")
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
