package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

const (
	CacheModeSet = "set"
	CacheModeGet = "get"
)

type CacheInfo struct {
	BrokerName  string      `yaml:"broker-name,omitempty" mapstructure:"broker-name,omitempty" json:"broker-name,omitempty"`
	Key         string      `yaml:"key,omitempty" mapstructure:"key,omitempty" json:"key,omitempty"`
	Mode        string      `yaml:"mode,omitempty" mapstructure:"mode,omitempty" json:"mode,omitempty"`
	OnCacheMiss OnCacheMiss `yaml:"on-cache-miss,omitempty" mapstructure:"on-cache-miss,omitempty" json:"on-cache-miss,omitempty"`
}

type OnCacheMiss struct {
	StatusCode           int    `yaml:"status-code,omitempty" mapstructure:"status-code,omitempty" json:"status-code,omitempty"`
	RefCacheMissResponse string `yaml:"ref-cachemiss-response,omitempty" mapstructure:"ref-cachemiss-response,omitempty" json:"ref-cachemiss-response,omitempty"`
}

type Response struct {
	Id                string          `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	Guard             string          `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
	RefSimpleResponse string          `yaml:"ref-simple-response,omitempty" mapstructure:"ref-simple-response,omitempty" json:"ref-simple-response,omitempty"`
	Headers           []NameValuePair `yaml:"headers,omitempty" json:"headers,omitempty" mapstructure:"headers,omitempty"`
	StatusCode        int             `yaml:"status-code,omitempty" mapstructure:"status-code,omitempty" json:"status-code,omitempty"`
	Cache             CacheInfo       `yaml:"cache,omitempty" mapstructure:"cache,omitempty" json:"cache,omitempty"`
}

type ResponseActivity struct {
	Activity  `yaml:",inline" json:",inline"`
	Responses []Response `yaml:"responses,omitempty" mapstructure:"responses,omitempty" json:"responses,omitempty"`
}

func (c *ResponseActivity) WithName(n string) *ResponseActivity {
	c.Nm = n
	return c
}

func (c *ResponseActivity) WithDescription(n string) *ResponseActivity {
	c.Cm = n
	return c
}

func (c *ResponseActivity) WithExpressionContext(n string) *ResponseActivity {
	c.ExprContextName = n
	return c
}

func NewResponseActivity() *ResponseActivity {
	a := ResponseActivity{}
	a.Tp = ResponseActivityType
	return &a
}

func NewResponseActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewResponseActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewResponseActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewResponseActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
