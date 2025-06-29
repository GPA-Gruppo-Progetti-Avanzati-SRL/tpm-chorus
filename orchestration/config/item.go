package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"strings"
	"time"
)

type Type string

const (
	RequestActivityType             Type = "request-activity"
	NopActivityType                 Type = "nop-activity"
	EchoActivityType                Type = "echo-activity"
	EndpointActivityType            Type = "rest-activity"
	ResponseActivityType            Type = "response-activity"
	KafkaActivityType               Type = "kafka-activity"
	NestedOrchestrationActivityType Type = "nested-orchestration-activity"
	MongoActivityType               Type = "mongo-activity"
	TransformActivityType           Type = "transform-activity"
	ScriptActivityType              Type = "script-activity"
	JsonSchemaActivityType          Type = "json-schema-activity"
	LoopActivityType                Type = "loop-activity"
	CacheActivityType               Type = "cache-activity"

	MongoDbActor    = "MongoDB"
	WebServiceActor = "WebService"
	XPAP            = "xPAP"
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
	NopActivityType:                 {Tp: NopActivityType, UnmarshallFromJSON: NewNopActivityFromJSON, UnmarshalFromYAML: NewNopActivityFromYAML},
	EndpointActivityType:            {Tp: EndpointActivityType, UnmarshallFromJSON: NewEndpointActivityFromJSON, UnmarshalFromYAML: NewEndpointActivityFromYAML},
	KafkaActivityType:               {Tp: KafkaActivityType, UnmarshallFromJSON: NewKafkaActivityFromJSON, UnmarshalFromYAML: NewKafkaActivityFromYAML},
	NestedOrchestrationActivityType: {Tp: NestedOrchestrationActivityType, UnmarshallFromJSON: NewNestedOrchestrationActivityFromJSON, UnmarshalFromYAML: NewNestedOrchestrationActivityFromYAML},
	MongoActivityType:               {Tp: MongoActivityType, UnmarshallFromJSON: NewMongoActivityFromJSON, UnmarshalFromYAML: NewMongoActivityFromYAML},
	TransformActivityType:           {Tp: TransformActivityType, UnmarshallFromJSON: NewTransformActivityFromJSON, UnmarshalFromYAML: NewTransformActivityFromYAML},
	ScriptActivityType:              {Tp: ScriptActivityType, UnmarshallFromJSON: NewScriptActivityFromJSON, UnmarshalFromYAML: NewScriptActivityFromYAML},
	JsonSchemaActivityType:          {Tp: JsonSchemaActivityType, UnmarshallFromJSON: NewJsonSchemaActivityFromJSON, UnmarshalFromYAML: NewJsonSchemaActivityFromYAML},
	LoopActivityType:                {Tp: LoopActivityType, UnmarshallFromJSON: NewLoopActivityFromJSON, UnmarshalFromYAML: NewLoopActivityFromYAML},
	CacheActivityType:               {Tp: CacheActivityType, UnmarshallFromJSON: NewCacheActivityFromJSON, UnmarshalFromYAML: NewCacheActivityFromYAML},
}

type Guarded interface {
	IsGuarded() string
}

type Configurable interface {
	Name() string
	Type() Type
	Enabled() string
	ActorWithDefault(defActor string) string
	Actor() string
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
	ActivityMetricsGroupId = "activity"
	DefaultMetricsGroupId  = "activity"
	DefaultCounterId       = "activity-counter"
	DefaultHistogramId     = "activity-duration"

	DefaultActivityBoundary = "global"
)

var DefaultMetricsCfg = promutil.MetricsConfigReference{
	GId:         DefaultMetricsGroupId,
	CounterId:   DefaultCounterId,
	HistogramId: DefaultHistogramId,
}

//go:embed activity-metrics.yml
var activityMetrics []byte

func ActivityMetrics(groupId string) (promutil.MetricGroupConfig, error) {
	var cfg promutil.MetricGroupConfig
	err := yaml.Unmarshal(activityMetrics, &cfg)
	if err != nil {
		return promutil.MetricGroupConfig{}, err
	}

	cfg.GroupId = groupId
	return cfg, nil
}

func MustActivityMetrics(groupId string) promutil.MetricGroupConfig {
	const semLogContext = "activity::metrics-configs"
	cfg, err := ActivityMetrics(ActivityMetricsGroupId)
	if err != nil {
		log.Fatal().Err(err).Str("group-id", ActivityMetricsGroupId).Msg(semLogContext)
	}

	return cfg
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
	Actr            string                          `yaml:"actor,omitempty" mapstructure:"actor,omitempty" json:"actor,omitempty"`
	BndryName       string                          `yaml:"boundary-name,omitempty" mapstructure:"boundary-name,omitempty" json:"boundary-name,omitempty"`
	BndryFlag       bool                            `yaml:"is-boundary,omitempty" mapstructure:"is-boundary,omitempty" json:"is-boundary,omitempty"`
	ProcessVars     []ProcessVar                    `yaml:"process-vars,omitempty" mapstructure:"process-vars,omitempty" json:"process-vars,omitempty"`
	En              string                          `yaml:"enabled,omitempty" mapstructure:"enabled,omitempty" json:"enabled,omitempty"`
	EstimatedTim    string                          `yaml:"estimated-time,omitempty" mapstructure:"estimated-time,omitempty" json:"estimated-time,omitempty"`
	MetricsCfg      promutil.MetricsConfigReference `yaml:"ref-metrics,omitempty" mapstructure:"ref-metrics,omitempty" json:"ref-metrics,omitempty"`
	Definition      string                          `yaml:"ref-definition,omitempty" mapstructure:"ref-definition,omitempty" json:"ref-definition,omitempty"`
	ExprContextName string                          `yaml:"input-source,omitempty" mapstructure:"input-source,omitempty" json:"input-source,omitempty"`
}

func (c *Activity) Dup(newName string) Activity {

	var pv []ProcessVar
	if len(c.ProcessVars) > 0 {
		pv = make([]ProcessVar, len(c.ProcessVars))
		_ = copy(pv, c.ProcessVars)
	}

	actNew := Activity{
		Nm:              newName,
		Tp:              c.Tp,
		Cm:              c.Cm,
		Actr:            c.Actr,
		BndryName:       c.BndryName,
		BndryFlag:       c.BndryFlag,
		EstimatedTim:    c.EstimatedTim,
		ProcessVars:     pv,
		En:              c.En,
		MetricsCfg:      c.MetricsCfg,
		Definition:      c.Definition,
		ExprContextName: c.ExprContextName,
	}

	return actNew
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

func (c *Activity) ActorWithDefault(defActor string) string {
	if c.Actr == "" {
		return defActor
	}

	return c.Actr
}

func (c *Activity) Actor() string {
	return c.Actr
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

func (c *Activity) EstimatedTime() time.Duration {
	return util.ParseDuration(c.EstimatedTim, time.Duration(0))
}

func (c *Activity) WfCaseDeadlineExceeded(currentTiming, reqDeadline time.Duration) error {
	const semLogContext = "activity::wfc-deadline-exceeded"
	activityEstimatedTime := c.EstimatedTime()
	if reqDeadline != 0 {
		if currentTiming+activityEstimatedTime > reqDeadline {
			err := errors.New("request deadline exceeded")
			log.Error().Err(err).Float64("ete.s", activityEstimatedTime.Seconds()).Float64("wfc-timing.s", currentTiming.Seconds()).Float64("deadline.s", reqDeadline.Seconds()).Msg(semLogContext)
			return err
		}
	}

	return nil
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
	ErrorLevel  string `yaml:"error-level,omitempty" mapstructure:"error-level,omitempty" json:"error-level,omitempty"`
}

func (ei ErrorInfo) IsZero() bool {
	return ei.StatusCode == 0 && ei.Ambit == "" && ei.Message == ""
}

func (ei ErrorInfo) IsGuarded() string {
	return ei.Guard
}

const (
	XFormMerge         = "merge"
	XFormTemplate      = "template"
	XFormKazaam        = "kazaam"
	XFormKazaamDynamic = "kazaam-dynamic"
	XFormJsonExt2Json  = "jsonext2json"
)

// OnResponseAction TODO Verificare dove vengono utilizzate le transforms.
type OnResponseAction struct {
	StatusCode                              int                          `yaml:"status-code,omitempty" mapstructure:"status-code,omitempty" json:"status-code,omitempty"`
	IgnoreNonApplicationJsonResponseContent bool                         `yaml:"ignore-non-json-response-body,omitempty" json:"ignore-non-json-response-body,omitempty" mapstructure:"ignore-non-json-response-body,omitempty"`
	ProcessVars                             []ProcessVar                 `yaml:"process-vars,omitempty" mapstructure:"process-vars,omitempty" json:"process-vars,omitempty"`
	Errors                                  []ErrorInfo                  `yaml:"error,omitempty" mapstructure:"error,omitempty" json:"error,omitempty"`
	Transforms                              []kzxform.TransformReference `yaml:"transforms,omitempty" mapstructure:"transforms,omitempty" json:"transforms,omitempty"`
	Properties                              map[string]string            `yaml:"properties,omitempty" mapstructure:"properties,omitempty" json:"properties,omitempty"` // activity dependent properties
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
