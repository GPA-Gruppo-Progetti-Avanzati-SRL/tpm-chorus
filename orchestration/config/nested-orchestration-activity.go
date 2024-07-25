package config

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
)

type NestedOrchestrationActivity struct {
	Activity
	Message string `yaml:"message,omitempty" mapstructure:"message,omitempty" json:"message,omitempty"`
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
	c.ExprScope = n
	return c
}

func NewNestedOrchestrationActivity() *NestedOrchestrationActivity {
	s := NestedOrchestrationActivity{}
	s.Tp = NestedOrchestrationActivityType
	return &s
}

func NewNestedOrchestrationActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewEchoActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewNestedOrchestrationActivityFromYAML(mp interface{}) (Configurable, error) {
	sa := NewEchoActivity()
	err := mapstructure.Decode(mp, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
