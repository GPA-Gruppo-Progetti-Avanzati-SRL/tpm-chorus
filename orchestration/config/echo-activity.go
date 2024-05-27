package config

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
)

type EchoActivity struct {
	Activity
	Message string `yaml:"message,omitempty" mapstructure:"message,omitempty" json:"message,omitempty"`
}

func (c *EchoActivity) WithName(n string) *EchoActivity {
	c.Nm = n
	return c
}

func (c *EchoActivity) WithDescription(n string) *EchoActivity {
	c.Cm = n
	return c
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

func NewEchoActivityFromYAML(mp interface{}) (Configurable, error) {
	sa := NewEchoActivity()
	err := mapstructure.Decode(mp, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
