package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

const (
	PersonallyIdentifiableInformationDefaultDomain    = "undef"
	PersonallyIdentifiableInformationDefaultAppliesTo = "none"
)

type PersonallyIdentifiableInformation struct {
	Domain    string `json:"domain,omitempty" yaml:"domain,omitempty" mapstructure:"domain,omitempty"`
	AppliesTo string `json:"applies-to,omitempty" yaml:"applies-to,omitempty" mapstructure:"applies-to,omitempty"`
}

func (pii PersonallyIdentifiableInformation) IsZero() bool {
	return pii.Domain == "" && pii.AppliesTo == ""
}

func (pii *PersonallyIdentifiableInformation) Initialize() {
	if pii.IsZero() {
		pii.Domain = PersonallyIdentifiableInformationDefaultDomain
		pii.AppliesTo = PersonallyIdentifiableInformationDefaultAppliesTo
	}
}

type Endpoint struct {
	Id          string                            `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	Name        string                            `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Description string                            `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	Definition  string                            `yaml:"ref-definition,omitempty" mapstructure:"ref-definition,omitempty" json:"ref-definition,omitempty"`
	PII         PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
}

type EndpointActivity struct {
	Activity  `yaml:",inline" json:",inline"`
	Endpoints []Endpoint `yaml:"endpoints,omitempty" mapstructure:"endpoints,omitempty" json:"endpoints,omitempty"`
}

func (c *EndpointActivity) WithName(n string) *EndpointActivity {
	c.Nm = n
	return c
}

func (c *EndpointActivity) WithDescription(n string) *EndpointActivity {
	c.Cm = n
	return c
}

func (c *EndpointActivity) WithExpressionContext(n string) *EndpointActivity {
	c.ExprContextName = n
	return c
}

func NewEndpointActivity() *EndpointActivity {
	s := EndpointActivity{}
	s.Tp = EndpointActivityType
	return &s
}

func NewEndpointActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewEndpointActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	for k := range i.Endpoints {
		i.Endpoints[k].PII.Initialize()
	}

	return i, nil
}

func NewEndpointActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	epa := NewEndpointActivity()
	// err := mapstructure.Decode(mp, epa)
	err := yaml.Unmarshal(b, epa)
	if err != nil {
		return nil, err
	}

	for k := range epa.Endpoints {
		epa.Endpoints[k].PII.Initialize()
	}

	return epa, nil
}
