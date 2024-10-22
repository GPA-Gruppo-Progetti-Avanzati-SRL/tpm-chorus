package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

// RequestValidation
// 20220726 Validations used to be a simple string. now it's a  struct with a few fields to be used.
type RequestValidation struct {
	Name        string `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Description string `yaml:"descr,omitempty" mapstructure:"descr,omitempty" json:"descr,omitempty"`
	Expr        string `yaml:"expr,omitempty" mapstructure:"expr,omitempty" json:"expr,omitempty"`
}

type RequestActivity struct {
	Activity `yaml:",inline" json:",inline"`
	// ProcessVars []ProcessVar `yaml:"process-vars,omitempty" mapstructure:"process-vars,omitempty" json:"process-vars,omitempty"`
	Validations []RequestValidation `yaml:"validations,omitempty" mapstructure:"validations,omitempty" json:"validations,omitempty"`
}

func (c *RequestActivity) WithName(n string) *RequestActivity {
	c.Nm = n
	return c
}

func (c *RequestActivity) WithActor(n string) *RequestActivity {
	c.Actr = n
	return c
}

func (c *RequestActivity) WithDescription(n string) *RequestActivity {
	c.Cm = n
	return c
}

func NewRequestActivity() *RequestActivity {
	s := RequestActivity{}
	s.Tp = RequestActivityType
	return &s
}

func NewRequestActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewRequestActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewRequestActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewRequestActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
