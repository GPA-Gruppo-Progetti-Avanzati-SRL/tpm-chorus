package plconfig

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

const (
	SinkTypeKafka        = "kafka"
	SinkTypeRawKafka     = "kafka-raw"
	MessageKeyHeaderName = "X-Kafka-Key"

	// from pipeline package not to have cyclic issues

)

type KafkaSinkBodyConfig struct {
	Typ           string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"`
	ExternalValue string `yaml:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
	Data          []byte `yaml:"data,omitempty" json:"data,omitempty" mapstructure:"data,omitempty"`
}

func (pd KafkaSinkBodyConfig) IsZero() bool {
	return len(pd.Data) == 0 && pd.ExternalValue == ""
}

type KafkaSinkNameValuePair struct {
	Name  string `mapstructure:"name,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	Value string `mapstructure:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty"`
}

type KafkaSinkEventConfig struct {
	Key     string                   `mapstructure:"key,omitempty" json:"key,omitempty" yaml:"key,omitempty"`
	Body    KafkaSinkBodyConfig      `mapstructure:"body,omitempty" json:"body,omitempty" yaml:"body,omitempty"`
	Headers []KafkaSinkNameValuePair `mapstructure:"headers,omitempty" json:"headers,omitempty" yaml:"headers,omitempty"`
}

type KafkaSinkDefinition struct {
	BrokerName                string                                `yaml:"broker-name,omitempty" mapstructure:"broker-name,omitempty" json:"broker-name,omitempty"`
	TopicName                 string                                `json:"topic-name,omitempty" yaml:"topic-name,omitempty" mapstructure:"topic-name,omitempty"`
	Event                     KafkaSinkEventConfig                  `json:"event,omitempty" yaml:"event,omitempty" mapstructure:"event,omitempty"`
	RefMetrics                *promutil.MetricsConfigReference      `yaml:"ref-metrics,omitempty"  mapstructure:"ref-metrics,omitempty"  json:"ref-metrics,omitempty"`
	OnNotCompleted            KafkaSinkOnNotCompleted               `yaml:"on-not-completed,omitempty"  mapstructure:"on-not-completed,omitempty"  json:"on-not-completed,omitempty"`
	WithSynchDlv              string                                `yaml:"with-synch-delivery,omitempty" mapstructure:"with-synch-delivery,omitempty" json:"with-synch-delivery,omitempty"`
	FlushConfig               KafkaSinkStageMessageQueueFlushConfig `yaml:"flush-config,omitempty"  mapstructure:"flush-config,omitempty" json:"flush-config,omitempty"`
	WithRandomError           int                                   `yaml:"with-random-error,omitempty" mapstructure:"with-random-error,omitempty" json:"with-random-error,omitempty"`
	MessageProducerBufferSize int                                   `yaml:"mp-buffer-size,omitempty" mapstructure:"mp-buffer-size,omitempty" json:"mp-buffer-size,omitempty"`
	// Guard      string               `json:"guard" yaml:"guard" mapstructure:"guard"`
}

func (def *KafkaSinkDefinition) WriteToFile(folderName string, fileName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "kafka-sink-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(def)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	err = fileutil.WriteFile(fn, b, os.ModePerm, writeOpts...)
	//outFileName, _ := fileutil.ResolvePath(fn)
	//err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func (d *KafkaSinkDefinition) WithSynchDelivery() bool {
	if d.WithSynchDlv == "" || d.WithSynchDlv == "true" {
		return true
	}

	return false
}

func (def *KafkaSinkDefinition) WaitConfigOnNotCompleted() KafkaSinkOnNotCompleted {
	onc := def.OnNotCompleted
	if onc.IsZero() {
		return onc
	}

	if onc.FirstWait == 0 {
		onc.FirstWait = onc.Wait
	}

	if onc.MaxWaitTimes <= 0 {
		onc.MaxWaitTimes = 1
	}

	return onc
}

type KafkaSinkOnNotCompleted struct {
	FirstWait    time.Duration `yaml:"first-wait,omitempty"  mapstructure:"first-wait,omitempty"  json:"first-wait,omitempty"`
	Wait         time.Duration `yaml:"wait,omitempty"  mapstructure:"wait,omitempty"  json:"wait,omitempty"`
	MaxWaitTimes int           `yaml:"max-wait-times,omitempty"  mapstructure:"max-wait-times,omitempty"  json:"max-wait-times,omitempty"`
}

type KafkaSinkStageMessageQueueFlushConfig struct {
	FlushTimeoutMs             int `yaml:"prd-flush-timeout-ms,omitempty"  json:"prd-flush-timeout-ms,omitempty" mapstructure:"prd-flush-timeout-ms,omitempty"`
	FlushStride                int `yaml:"prd-flush-stride,omitempty"  json:"prd-flush-stride,omitempty" mapstructure:"prd-flush-stride,omitempty"`
	OnCompletionFlushTimeoutMs int `yaml:"onc-flush-timeout-ms,omitempty"  json:"onc-flush-timeout-ms,omitempty" mapstructure:"onc-flush-timeout-ms,omitempty"`
}

func (cfg KafkaSinkStageMessageQueueFlushConfig) FlushWhenProducingToTopics(messageNum int) bool {
	b := cfg.FlushTimeoutMs != 0 && cfg.FlushStride != 0 && messageNum%cfg.FlushStride == 0
	return b
}

func (cfg KafkaSinkStageMessageQueueFlushConfig) FlushOnCompletion() bool {
	return cfg.OnCompletionFlushTimeoutMs != 0
}

func (onc KafkaSinkOnNotCompleted) IsZero() bool {
	return onc.Wait <= 0
}
