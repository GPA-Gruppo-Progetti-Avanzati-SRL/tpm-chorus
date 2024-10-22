package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type EchoActivity struct {
	Activity `yaml:",inline" json:",inline"`
	Message  string `yaml:"message,omitempty" mapstructure:"message,omitempty" json:"message,omitempty"`
}

func (c *EchoActivity) WithName(n string) *EchoActivity {
	c.Nm = n
	return c
}

func (c *EchoActivity) WithActor(n string) *EchoActivity {
	c.Actr = n
	return c
}

func (c *EchoActivity) WithDescription(n string) *EchoActivity {
	c.Cm = n
	return c
}

func (c *EchoActivity) WithExpressionContext(n string) *EchoActivity {
	c.ExprContextName = n
	return c
}

func (c *EchoActivity) WithRefDefinition(n string) *EchoActivity {
	c.Definition = n
	return c
}

func (c *EchoActivity) Dup(newName string) *EchoActivity {
	actNew := EchoActivity{
		Activity: c.Activity.Dup(newName),
		Message:  c.Message,
	}

	return &actNew
}

func NewEchoActivity() *EchoActivity {
	s := EchoActivity{}
	s.Tp = EchoActivityType
	return &s
}

func NewEchoActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewEchoActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewEchoActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewEchoActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
