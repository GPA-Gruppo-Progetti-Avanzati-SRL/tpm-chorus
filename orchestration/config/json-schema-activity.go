package config

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/jsonschemaregistry"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type JsonSchemaActivity struct {
	Activity `yaml:",inline" json:",inline"`
}

func (c *JsonSchemaActivity) WithName(n string) *JsonSchemaActivity {
	c.Nm = n
	return c
}

func (c *JsonSchemaActivity) WithDescription(n string) *JsonSchemaActivity {
	c.Cm = n
	return c
}

func (c *JsonSchemaActivity) WithExpressionContext(n string) *JsonSchemaActivity {
	c.ExprContextName = n
	return c
}

func (c *JsonSchemaActivity) WithRefDefinition(n string) *JsonSchemaActivity {
	c.Definition = n
	return c
}

func NewJsonSchemaActivity() *JsonSchemaActivity {
	s := JsonSchemaActivity{}
	s.Tp = JsonSchemaActivityType
	return &s
}

func NewJsonSchemaActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewJsonSchemaActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewJsonSchemaActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewJsonSchemaActivity()
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}

type JsonSchemaActivityDefinition struct {
	SchemaRef         string            `yaml:"schema-file,omitempty" json:"schema-file,omitempty" mapstructure:"schema-file,omitempty"`
	OnResponseActions OnResponseActions `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func (def *JsonSchemaActivityDefinition) IsZero() bool {
	return def.SchemaRef == ""
}

func UnmarshalJsonSchemaActivityDefinition(schemaNamespace, def string, refs DataReferences) (JsonSchemaActivityDefinition, error) {
	const semLogContext = "json-schema-activity-definition::unmarshal"

	var err error
	maDef := JsonSchemaActivityDefinition{}

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

	if maDef.SchemaRef != "" {
		data, ok := refs.Find(maDef.SchemaRef)
		if len(data) == 0 || !ok {
			err = errors.New("cannot find schema file")
			log.Error().Err(err).Str("def", maDef.SchemaRef).Msg(semLogContext)
			return maDef, err
		}

		err = jsonschemaregistry.Register(schemaNamespace, maDef.SchemaRef, data)
		if err != nil {
			log.Error().Err(err).Str("def", maDef.SchemaRef).Msg(semLogContext)
			return maDef, err
		}
	}

	return maDef, nil
}
