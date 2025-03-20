package kafkasink

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	SinkTypeKafka        = "kafka"
	MessageKeyHeaderName = "X-Kafka-Key"
)

type BodyConfig struct {
	Typ           string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"`
	ExternalValue string `yaml:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
	Data          []byte `yaml:"data,omitempty" json:"data,omitempty" mapstructure:"data,omitempty"`
}

func (pd BodyConfig) IsZero() bool {
	return len(pd.Data) == 0 && pd.ExternalValue == ""
}

type NameValuePair struct {
	Name  string `mapstructure:"name,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	Value string `mapstructure:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty"`
}

type KafkaSinkEventConfig struct {
	Key     string          `mapstructure:"key,omitempty" json:"key,omitempty" yaml:"key,omitempty"`
	Body    BodyConfig      `mapstructure:"body,omitempty" json:"body,omitempty" yaml:"body,omitempty"`
	Headers []NameValuePair `mapstructure:"headers,omitempty" json:"headers,omitempty" yaml:"headers,omitempty"`
}

type KafkaSinkDefinition struct {
	BrokerName string `yaml:"broker-name,omitempty" mapstructure:"broker-name,omitempty" json:"broker-name,omitempty"`
	TopicName  string `json:"topic-name" yaml:"topic-name" mapstructure:"topic-name"`
	// Guard      string               `json:"guard" yaml:"guard" mapstructure:"guard"`
	Event KafkaSinkEventConfig `json:"event" yaml:"event" mapstructure:"event"`
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
