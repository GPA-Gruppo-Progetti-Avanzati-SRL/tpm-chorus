package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"strings"
	"time"
)

type Type string

const (
	RequestActivityType             Type = "request-activity"
	EchoActivityType                Type = "echo-activity"
	EndpointActivityType            Type = "rest-activity"
	ResponseActivityType            Type = "response-activity"
	KafkaActivityType               Type = "kafka-activity"
	NestedOrchestrationActivityType Type = "nested-orchestration-activity"
	MongoActivityType               Type = "mongo-activity"
	TransformActivityType           Type = "transform-activity"
	ScriptActivityType              Type = "script-activity"
)

type ActivityTypeRegistryEntry struct {
	Tp                 Type
	UnmarshallFromJSON func(raw json.RawMessage) (Configurable, error)
	UnmarshalFromYAML  func(b []byte /* mp interface{} */) (Configurable, error)
}

var activityTypeRegistry = map[Type]ActivityTypeRegistryEntry{
	RequestActivityType:             {Tp: RequestActivityType, UnmarshallFromJSON: NewRequestActivityFromJSON, UnmarshalFromYAML: NewRequestActivityFromYAML},
	ResponseActivityType:            {Tp: ResponseActivityType, UnmarshallFromJSON: NewResponseActivityFromJSON, UnmarshalFromYAML: NewResponseActivityFromYAML},
	EchoActivityType:                {Tp: EchoActivityType, UnmarshallFromJSON: NewEchoActivityFromJSON, UnmarshalFromYAML: NewEchoActivityFromYAML},
	EndpointActivityType:            {Tp: EndpointActivityType, UnmarshallFromJSON: NewEndpointActivityFromJSON, UnmarshalFromYAML: NewEndpointActivityFromYAML},
	KafkaActivityType:               {Tp: KafkaActivityType, UnmarshallFromJSON: NewKafkaActivityFromJSON, UnmarshalFromYAML: NewKafkaActivityFromYAML},
	NestedOrchestrationActivityType: {Tp: NestedOrchestrationActivityType, UnmarshallFromJSON: NewNestedOrchestrationActivityFromJSON, UnmarshalFromYAML: NewNestedOrchestrationActivityFromYAML},
	MongoActivityType:               {Tp: MongoActivityType, UnmarshallFromJSON: NewMongoActivityFromJSON, UnmarshalFromYAML: NewMongoActivityFromYAML},
	TransformActivityType:           {Tp: TransformActivityType, UnmarshallFromJSON: NewTransformActivityFromJSON, UnmarshalFromYAML: NewTransformActivityFromYAML},
	ScriptActivityType:              {Tp: ScriptActivityType, UnmarshallFromJSON: NewScriptActivityFromJSON, UnmarshalFromYAML: NewScriptActivityFromYAML},
}

type Guarded interface {
	IsGuarded() string
}

type Configurable interface {
	Name() string
	Type() Type
	Enabled() string
	Boundary() string
	IsBoundary() bool
	Description() string
	RefDefinition() string
	MetricsConfig() promutil.MetricsConfigReference
	ExpressionContextNameStringReference() string
}

func NewActivityFromJSON(t Type, message json.RawMessage) (Configurable, error) {

	if e, ok := activityTypeRegistry[t]; ok {
		c, err := e.UnmarshallFromJSON(message)
		return c, err
	}

	return nil, fmt.Errorf("unknown activity type %s", t)
}

func NewActivityFromYAML(t Type, b []byte /* m interface{} */) (Configurable, error) {

	if e, ok := activityTypeRegistry[t]; ok {
		c, err := e.UnmarshalFromYAML(b)
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

type ActivityProperty struct {
	Name     string `yaml:"name,omitempty" json:"name,omitempty" mapstructure:"name,omitempty"`
	Typ      string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"`
	Value    string `yaml:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
	ExtValue string `yaml:"external-value,omitempty" json:"external,omitempty" mapstructure:"external,omitempty"`
}

func (ap ActivityProperty) IsValid() error {

	var err error
	if ap.Value != "" && ap.ExtValue != "" {
		err = errors.New("only one of value or external-value can be specified")
	} else if ap.Value == "" && ap.ExtValue == "" {
		err = errors.New("one of value and external-value should be specified")
	}

	return err
}

type Activity struct {
	Nm              string                          `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Tp              Type                            `yaml:"type,omitempty" mapstructure:"type,omitempty" json:"type,omitempty"`
	Cm              string                          `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	BndryName       string                          `yaml:"boundary-name,omitempty" mapstructure:"boundary-name,omitempty" json:"boundary-name,omitempty"`
	BndryFlag       bool                            `yaml:"is-boundary,omitempty" mapstructure:"is-boundary,omitempty" json:"is-boundary,omitempty"`
	ProcessVars     []ProcessVar                    `yaml:"process-vars,omitempty" mapstructure:"process-vars,omitempty" json:"process-vars,omitempty"`
	En              string                          `yaml:"enabled,omitempty" mapstructure:"enabled,omitempty" json:"enabled,omitempty"`
	MetricsCfg      promutil.MetricsConfigReference `yaml:"ref-metrics,omitempty" mapstructure:"ref-metrics,omitempty" json:"ref-metrics,omitempty"`
	Definition      string                          `yaml:"ref-definition,omitempty" mapstructure:"ref-definition,omitempty" json:"ref-definition,omitempty"`
	ExprContextName string                          `yaml:"input-source,omitempty" mapstructure:"input-source,omitempty" json:"input-source,omitempty"`
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

func (c *Activity) RefDefinition() string {
	return c.Definition
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

// ActivityDefaultExpressionContextName To be aligned with wfcase.InitialRequestHarEntryId
const (
	ActivityDefaultExpressionContextName = "request"
)

func (c *Activity) ExpressionContextNameStringReference() string {
	if c.ExprContextName == "" {
		return ActivityDefaultExpressionContextName
	}

	return c.ExprContextName
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

type ErrorInfo struct {
	StatusCode  int    `yaml:"status-code,omitempty" mapstructure:"status-code,omitempty" json:"status-code,omitempty"`
	Ambit       string `yaml:"ambit,omitempty" mapstructure:"ambit,omitempty" json:"ambit,omitempty"`
	Message     string `yaml:"message,omitempty" mapstructure:"message,omitempty" json:"message,omitempty"`
	Code        string `yaml:"code,omitempty" mapstructure:"code,omitempty" json:"code,omitempty"`
	Step        string `yaml:"step,omitempty" mapstructure:"step,omitempty" json:"step,omitempty"`
	Description string `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	Guard       string `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
}

func (ei ErrorInfo) IsZero() bool {
	return ei.StatusCode == 0 && ei.Ambit == "" && ei.Message == ""
}

func (ei ErrorInfo) IsGuarded() string {
	return ei.Guard
}

const (
	XFormMerge    = "merge"
	XFormTemplate = "template"
	XFormKazaam   = "kazaam"
)

// OnResponseAction TODO Verificare dove vengono utilizzate le transforms.
type OnResponseAction struct {
	StatusCode                              int                            `yaml:"status-code,omitempty" mapstructure:"status-code,omitempty" json:"status-code,omitempty"`
	IgnoreNonApplicationJsonResponseContent bool                           `yaml:"ignore-non-json-response-body,omitempty" json:"ignore-non-json-response-body,omitempty" mapstructure:"ignore-non-json-response-body,omitempty"`
	ProcessVars                             []ProcessVar                   `yaml:"process-vars,omitempty" mapstructure:"process-vars,omitempty" json:"process-vars,omitempty"`
	Errors                                  []ErrorInfo                    `yaml:"error,omitempty" mapstructure:"error,omitempty" json:"error,omitempty"`
	Transforms                              []transform.TransformReference `yaml:"transforms,omitempty" mapstructure:"transforms,omitempty" json:"transforms,omitempty"`
	Properties                              map[string]string              `yaml:"properties,omitempty" mapstructure:"properties,omitempty" json:"properties,omitempty"` // activity dependent properties
}

type OnResponseActions []OnResponseAction

func (acts OnResponseActions) FindByStatusCode(statusCode int) int {

	matchedAction := -1
	defaultAction := -1
	for ndx, act := range acts {
		if act.StatusCode == statusCode {
			matchedAction = ndx
			break
		}

		if act.StatusCode == -1 {
			defaultAction = ndx
		}
	}

	if matchedAction < 0 && defaultAction >= 0 {
		matchedAction = defaultAction
	}

	return matchedAction
}

type CacheConfig struct {
	Key string `yaml:"key,omitempty" mapstructure:"key,omitempty" json:"key,omitempty"`
	// Mode             string                         `yaml:"mode,omitempty" mapstructure:"mode,omitempty" json:"mode,omitempty"`
	Namespace        string                         `json:"namespace,omitempty" yaml:"namespace,omitempty" mapstructure:"namespace,omitempty"`
	Ttl              time.Duration                  `yaml:"ttl,omitempty" mapstructure:"ttl,omitempty" json:"ttl,omitempty"`
	LinkedServiceRef cachelks.CacheLinkedServiceRef `yaml:"broker,omitempty" mapstructure:"broker,omitempty" json:"broker,omitempty"`
}

func (edcc *CacheConfig) IsZero() bool {
	return edcc.Key == "" && edcc.Namespace == "" && edcc.LinkedServiceRef.IsZero() && edcc.Ttl == 0
}

func (edcc *CacheConfig) Enabled() (bool, error) {

	if edcc.IsZero() {
		return false, nil
	}

	var sb strings.Builder

	numErrors := 0
	if edcc.Key == "" {
		sb.WriteString("missing cache key")
		numErrors++
	}

	if edcc.LinkedServiceRef.Typ == "" || edcc.LinkedServiceRef.Name == "" {
		if numErrors > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("missing linked service reference info")
		numErrors++
	}

	if numErrors > 0 {
		return false, errors.New(sb.String())
	}

	return true, nil
}
