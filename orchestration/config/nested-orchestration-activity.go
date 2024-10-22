package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type NestedOrchestrationActivity struct {
	Activity `yaml:",inline" json:",inline"`
}

func (c *NestedOrchestrationActivity) WithName(n string) *NestedOrchestrationActivity {
	c.Nm = n
	return c
}

func (c *NestedOrchestrationActivity) WithDescription(n string) *NestedOrchestrationActivity {
	c.Cm = n
	return c
}

func (c *NestedOrchestrationActivity) WithExpressionContext(n string) *NestedOrchestrationActivity {
	c.ExprContextName = n
	return c
}

func (c *NestedOrchestrationActivity) WithRefDefinition(n string) *NestedOrchestrationActivity {
	c.Definition = n
	return c
}

func (c *NestedOrchestrationActivity) Dup(newName string) *NestedOrchestrationActivity {
	actNew := NestedOrchestrationActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

func NewNestedOrchestrationActivity() *NestedOrchestrationActivity {
	s := NestedOrchestrationActivity{}
	s.Tp = NestedOrchestrationActivityType
	return &s
}

func NewNestedOrchestrationActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewNestedOrchestrationActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewNestedOrchestrationActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewNestedOrchestrationActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
