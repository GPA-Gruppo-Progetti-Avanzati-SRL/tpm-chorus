package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type LoopActivity struct {
	Activity `yaml:",inline" json:",inline"`
	PII      PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
}

func (c *LoopActivity) WithName(n string) *LoopActivity {
	c.Nm = n
	return c
}

func (c *LoopActivity) WithDescription(n string) *LoopActivity {
	c.Cm = n
	return c
}

func (c *LoopActivity) WithExpressionContext(n string) *LoopActivity {
	c.ExprContextName = n
	return c
}

func (c *LoopActivity) WithRefDefinition(n string) *LoopActivity {
	c.Definition = n
	return c
}

func (c *LoopActivity) Dup(newName string) *LoopActivity {
	actNew := LoopActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

func NewLoopActivity() *LoopActivity {
	s := LoopActivity{}
	s.Tp = LoopActivityType
	return &s
}

func NewLoopActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewLoopActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewLoopActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewLoopActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
