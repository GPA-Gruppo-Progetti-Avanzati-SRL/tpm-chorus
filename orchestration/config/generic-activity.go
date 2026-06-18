package config

import (
	"encoding/json"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"gopkg.in/yaml.v3"
)

type GenericActivity struct {
	Activity `yaml:",inline" json:",inline"`
}

func (c *GenericActivity) WithName(n string) *GenericActivity {
	c.Nm = n
	return c
}

func (c *GenericActivity) WithDescription(n string) *GenericActivity {
	c.Cm = n
	return c
}

func (c *GenericActivity) Dup(newName string) *GenericActivity {
	actNew := GenericActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

/*
func GenericActivity() *GenericActivity {
	s := GenericActivity{}
	s.Tp = NopActivityType
	return &s
}
*/

func NewGenericActivity() *GenericActivity {
	s := GenericActivity{
		Activity: Activity{
			Nm: util.NewUUID(),
			Tp: NopActivityType,
			Cm: "nop activity",
		},
	}

	return &s
}

func NewGenericActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewGenericActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewGenericActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewGenericActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
