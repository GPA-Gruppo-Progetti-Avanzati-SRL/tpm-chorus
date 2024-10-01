package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type TransformActivity struct {
	Activity `yaml:",inline" json:",inline"`
	PII      PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
}

func (c *TransformActivity) WithName(n string) *TransformActivity {
	c.Nm = n
	return c
}

func (c *TransformActivity) WithDescription(n string) *TransformActivity {
	c.Cm = n
	return c
}

func (c *TransformActivity) WithRefDefinition(n string) *TransformActivity {
	c.Definition = n
	return c
}

func (c *TransformActivity) WithExpressionContext(n string) *TransformActivity {
	c.ExprContextName = n
	return c
}

func NewTransformActivity() *TransformActivity {
	s := TransformActivity{}
	s.Tp = TransformActivityType
	return &s
}

func NewTransformActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewTransformActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	i.PII.Initialize()
	return i, nil
}

func NewTransformActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewTransformActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	sa.PII.Initialize()
	return sa, nil
}
