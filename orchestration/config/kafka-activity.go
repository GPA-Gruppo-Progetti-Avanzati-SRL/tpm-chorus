package config

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
)

type ProducerDefinition struct {
	TopicName         string             `yaml:"topic-name,omitempty" json:"topic-name,omitempty" mapstructure:"topic-name,omitempty"`
	TraceOpName       string             `yaml:"trace-op-name,omitempty" json:"trace-op-name,omitempty" mapstructure:"trace-op-name,omitempty"`
	Headers           []NameValuePair    `yaml:"headers,omitempty" json:"headers,omitempty" mapstructure:"headers,omitempty"`
	Key               string             `yaml:"key,omitempty" json:"key,omitempty" mapstructure:"key,omitempty"`
	Body              PostData           `yaml:"body,omitempty" json:"body,omitempty" mapstructure:"body,omitempty"`
	OnResponseActions []OnResponseAction `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func (def *ProducerDefinition) WriteToFile(folderName string, fileName string) error {
	const semLogContext = "kafka-producer-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(def)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	outFileName, _ := fileutil.ResolvePath(fn)
	err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

type Producer struct {
	Id          string                            `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	Name        string                            `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Description string                            `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	Definition  string                            `yaml:"ref-definition,omitempty" mapstructure:"ref-definition,omitempty" json:"ref-definition,omitempty"`
	PII         PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
}

type KafkaActivity struct {
	Activity   `yaml:",inline" json:",inline"`
	BrokerName string     `mapstructure:"broker-name" json:"broker-name" yaml:"broker-name"`
	Producers  []Producer `yaml:"producers,omitempty" mapstructure:"producers,omitempty" json:"producers,omitempty"`
}

func (c *KafkaActivity) WithName(n string) *KafkaActivity {
	c.Nm = n
	return c
}

func (c *KafkaActivity) WithActor(n string) *KafkaActivity {
	c.Actr = n
	return c
}

func (c *KafkaActivity) WithDescription(n string) *KafkaActivity {
	c.Cm = n
	return c
}

func (c *KafkaActivity) WithExpressionContext(n string) *KafkaActivity {
	c.ExprContextName = n
	return c
}

func (c *KafkaActivity) Dup(newName string) *KafkaActivity {

	var prods []Producer
	if len(c.Producers) > 0 {
		_ = copy(prods, c.Producers)
	}

	actNew := KafkaActivity{
		Activity:   c.Activity.Dup(newName),
		BrokerName: c.BrokerName,
		Producers:  prods,
	}

	return &actNew
}

func NewKafkaActivity() *KafkaActivity {
	s := KafkaActivity{}
	s.Tp = KafkaActivityType
	return &s
}

func NewKafkaActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewKafkaActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	for k := range i.Producers {
		i.Producers[k].PII.Initialize()
	}

	return i, nil
}

func NewKafkaActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	epa := NewKafkaActivity()
	// err := mapstructure.Decode(mp, epa)
	err := yaml.Unmarshal(b, epa)
	if err != nil {
		return nil, err
	}

	for k := range epa.Producers {
		epa.Producers[k].PII.Initialize()
	}

	return epa, nil
}
