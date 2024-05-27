package config

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
)

type Type string

const (
	RequestActivityType  Type = "request-activity"
	EchoActivityType     Type = "echo-activity"
	EndpointActivityType Type = "rest-activity"
	ResponseActivityType Type = "response-activity"
	KafkaActivityType    Type = "kafka-activity"
)

type ActivityTypeRegistryEntry struct {
	Tp                 Type
	UnmarshallFromJSON func(raw json.RawMessage) (Configurable, error)
	UnmarshalFromYAML  func(mp interface{}) (Configurable, error)
}

var activityTypeRegistry = map[Type]ActivityTypeRegistryEntry{
	RequestActivityType:  {Tp: RequestActivityType, UnmarshallFromJSON: NewRequestActivityFromJSON, UnmarshalFromYAML: NewRequestActivityFromYAML},
	ResponseActivityType: {Tp: ResponseActivityType, UnmarshallFromJSON: NewResponseActivityFromJSON, UnmarshalFromYAML: NewResponseActivityFromYAML},
	EchoActivityType:     {Tp: EchoActivityType, UnmarshallFromJSON: NewEchoActivityFromJSON, UnmarshalFromYAML: NewEchoActivityFromYAML},
	EndpointActivityType: {Tp: EndpointActivityType, UnmarshallFromJSON: NewEndpointActivityFromJSON, UnmarshalFromYAML: NewEndpointActivityFromYAML},
	KafkaActivityType:    {Tp: KafkaActivityType, UnmarshallFromJSON: NewKafkaActivityFromJSON, UnmarshalFromYAML: NewKafkaActivityFromYAML},
}

type Configurable interface {
	Name() string
	Type() Type
	Enabled() string
	Boundary() string
	IsBoundary() bool
	Description() string
	MetricsConfig() promutil.MetricsConfigReference
}

func NewActivityFromJSON(t Type, message json.RawMessage) (Configurable, error) {

	if e, ok := activityTypeRegistry[t]; ok {
		c, err := e.UnmarshallFromJSON(message)
		return c, err
	}

	return nil, fmt.Errorf("unknown activity type %s", t)
}

func NewActivityFromYAML(t Type, m interface{}) (Configurable, error) {

	if e, ok := activityTypeRegistry[t]; ok {
		c, err := e.UnmarshalFromYAML(m)
		return c, err
	}

	return nil, fmt.Errorf("unknown activity type %s", t)
}

const (
	DefaultMetricsGroupId = "activity"
	DefaultCounterId      = "activity-counter"
	DefaultHistogramId    = "activity-duration"

	DefaultActivityBoundary = "global"
)

var DefaultMetricsCfg = promutil.MetricsConfigReference{
	GId:         DefaultMetricsGroupId,
	CounterId:   DefaultCounterId,
	HistogramId: DefaultHistogramId,
}

type Activity struct {
	Nm          string                          `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Tp          Type                            `yaml:"type,omitempty" mapstructure:"type,omitempty" json:"type,omitempty"`
	Cm          string                          `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	BndryName   string                          `yaml:"boundary-name,omitempty" mapstructure:"boundary-name,omitempty" json:"boundary-name,omitempty"`
	BndryFlag   bool                            `yaml:"is-boundary,omitempty" mapstructure:"is-boundary,omitempty" json:"is-boundary,omitempty"`
	ProcessVars []ProcessVar                    `yaml:"process-vars,omitempty" mapstructure:"process-vars,omitempty" json:"process-vars,omitempty"`
	En          string                          `yaml:"enabled,omitempty" mapstructure:"enabled,omitempty" json:"enabled,omitempty"`
	MetricsCfg  promutil.MetricsConfigReference `yaml:"ref-metrics,omitempty" mapstructure:"ref-metrics,omitempty" json:"ref-metrics,omitempty"`
}

func (c *Activity) WithName(n string) *Activity {
	c.Nm = n
	return c
}

func (c *Activity) WithDescription(n string) *Activity {
	c.Cm = n
	return c
}

func (c *Activity) Name() string {
	return c.Nm
}

func (c *Activity) Type() Type {
	return c.Tp
}

func (c *Activity) Description() string {
	return c.Cm
}

func (c *Activity) Enabled() string {
	return c.En
}

func (c *Activity) Boundary() string {
	if c.BndryName == "" {
		c.BndryName = DefaultActivityBoundary
	}
	return c.BndryName
}

func (c *Activity) IsBoundary() bool {
	return c.BndryFlag
}

func (c *Activity) MetricsConfig() promutil.MetricsConfigReference {
	r := promutil.CoalesceMetricsConfig(c.MetricsCfg, DefaultMetricsCfg)
	return r
}

/*
func (c *Activity) MetricsConfig() promutil.MetricsConfigReference {

	c.MetricsCfg = promutil.CoalesceMetricsConfig(c.MetricsCfg, DefaultMetricsCfg)
	gid := DefaultMetricsCfg
	if c.MetricsCfg.GId != "" {
		gid.GId = c.MetricsCfg.GId
	}

	if c.MetricsCfg.CounterId != "" {
		gid.CounterId = c.MetricsCfg.CounterId
	}

	if c.MetricsCfg.HistogramId != "" {
		gid.HistogramId = c.MetricsCfg.HistogramId
	}
	return gid
}
*/
