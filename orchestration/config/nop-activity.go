package config

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"gopkg.in/yaml.v3"
)

type NopActivity struct {
	Activity `yaml:",inline" json:",inline"`
}

func (c *NopActivity) WithName(n string) *NopActivity {
	c.Nm = n
	return c
}

func (c *NopActivity) WithDescription(n string) *NopActivity {
	c.Cm = n
	return c
}

func (c *NopActivity) Dup(newName string) *NopActivity {
	actNew := NopActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

/*
func NewNopActivity() *NopActivity {
	s := NopActivity{}
	s.Tp = NopActivityType
	return &s
}
*/

func NewNopActivity() *NopActivity {
	s := NopActivity{
		Activity: Activity{
			Nm: util.NewUUID(),
			Tp: NopActivityType,
			Cm: "nop activity",
		},
	}

	return &s
}

func NewNopActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewNopActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewNopActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewNopActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
