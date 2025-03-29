package plconfig

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/tprod"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
)

const (
	RhpPipelineDefinitionFileName = "tpm-rhapsody-pipeline.yml"
)

type RhpPipelineEventDefinitionType struct {
	ContentType string `json:"content-type,omitempty" yaml:"content-type,omitempty" mapstructure:"content-type,omitempty"`
	Schema      string `json:"schema,omitempty" yaml:"schema,omitempty" mapstructure:"schema,omitempty"`
}

func (p *RhpPipelineEventDefinitionType) IsZero() bool {
	if p.ContentType == "" && p.Schema == "" {
		return true
	}

	return false
}

type RhpPipelineEventDefinition struct {
	Key  RhpPipelineEventDefinitionType `json:"key,omitempty" yaml:"key,omitempty" mapstructure:"key,omitempty"`
	Body RhpPipelineEventDefinitionType `json:"body,omitempty" yaml:"body,omitempty" mapstructure:"body,omitempty"`
}

func (p *RhpPipelineEventDefinition) IsZero() bool {
	if p.Body.IsZero() && p.Key.IsZero() {
		return true
	}

	return false
}

type RhpPipelineDestination struct {
	SingStage string `json:"sink-stage,omitempty" yaml:"sink-stage,omitempty" mapstructure:"sink-stage,omitempty"`
	Guard     string `json:"guard,omitempty" yaml:"guard,omitempty" mapstructure:"guard,omitempty"`
}

type RhpPipelinePathDefinition struct {
	OrchestrationFolder string                     `json:"orchestration-folder,omitempty" yaml:"orchestration-folder,omitempty" mapstructure:"orchestration-folder,omitempty"`
	EventInfo           RhpPipelineEventDefinition `json:"event,omitempty" yaml:"event,omitempty" mapstructure:"event,omitempty"`
	Destinations        []RhpPipelineDestination   `json:"destinations,omitempty" yaml:"destinations,omitempty" mapstructure:"destinations,omitempty"`
}

func (p *RhpPipelinePathDefinition) IsZero() bool {
	if p.OrchestrationFolder == "" && len(p.Destinations) == 0 && p.EventInfo.IsZero() {
		return true
	}

	return false
}

type RhpPipelineDefinition struct {
	Id                           string                           `json:"id,omitempty" yaml:"id,omitempty" mapstructure:"id,omitempty"`
	En                           string                           `json:"enabled,omitempty" yaml:"enabled,omitempty" mapstructure:"enabled,omitempty"`
	Description                  string                           `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`
	BrokerName                   string                           `yaml:"broker-name,omitempty" mapstructure:"broker-name,omitempty" json:"broker-name,omitempty"`
	SourceTopic                  string                           `json:"source-topic,omitempty" yaml:"source-topic,omitempty" mapstructure:"source-topic,omitempty"`
	WorkMode                     string                           `yaml:"work-mode,omitempty" mapstructure:"work-mode,omitempty" json:"work-mode,omitempty"`
	MessageProducerBufferSize    int                              `yaml:"mp-buffer-size,omitempty" mapstructure:"mp-buffer-size,omitempty" json:"mp-buffer-size,omitempty"`
	NumPartitions                string                           `yaml:"num-partitions,omitempty" mapstructure:"num-partitions,omitempty" json:"num-partitions,omitempty"`
	OnErrors                     []tprod.OnErrorPolicy            `yaml:"on-errors,omitempty" mapstructure:"on-errors,omitempty" json:"on-errors,omitempty"`
	CommitMode                   string                           `yaml:"commit-mode,omitempty" mapstructure:"commit-mode,omitempty" json:"commit-mode,omitempty"`
	GroupId                      string                           `yaml:"consumer-group-id,omitempty" mapstructure:"consumer-group-id,omitempty" json:"consumer-group-id,omitempty"`
	ProducerId                   string                           `yaml:"producer-tx-id,omitempty" mapstructure:"producer-tx-id,omitempty" json:"producer-tx-id,omitempty"`
	MaxPollTimeout               int                              `yaml:"max-poll-timeout,omitempty" mapstructure:"max-poll-timeout,omitempty" json:"max-poll-timeout,omitempty"`
	MaxPollTimeoutMs             string                           `yaml:"max-poll-timeout-ms,omitempty" mapstructure:"max-poll-timeout-ms,omitempty" json:"max-poll-timeout-ms,omitempty"`
	TickInterval                 string                           `yaml:"tick-interval,omitempty" mapstructure:"tick-interval,omitempty" json:"tick-interval,omitempty"`
	RefMetrics                   *promutil.MetricsConfigReference `yaml:"ref-metrics"  mapstructure:"ref-metrics"  json:"ref-metrics"`
	SpanName                     string                           `yaml:"tracing-span-name,omitempty" mapstructure:"tracing-span-name,omitempty" json:"tracing-span-name,omitempty"`
	DeadLetterTopic              string                           `json:"dead-letter-topic,omitempty" yaml:"dead-letter-topic,omitempty" mapstructure:"dead-letter-topic,omitempty"`
	Paths                        []RhpPipelinePathDefinition      `json:"paths,omitempty" yaml:"paths,omitempty" mapstructure:"paths,omitempty"`
	Sinks                        []SinkStageDefinitionReference   `json:"sink-stages,omitempty" yaml:"sink-stages,omitempty" mapstructure:"sink-stages,omitempty"`
	WithSynchDelivery            string                           `yaml:"with-synch-delivery,omitempty" mapstructure:"with-synch-delivery,omitempty" json:"with-synch-delivery,omitempty"`
	NoAbortOnAsyncDeliveryFailed string                           `yaml:"no-abort-on-async-delivery-failed,omitempty" mapstructure:"no-abort-on-async-delivery-failed,omitempty" json:"no-abort-on-async-delivery-failed,omitempty"`
}

func (d *RhpPipelineDefinition) GetMaxPollTimeoutMsAsInt() int {
	const semLogContext = "pipeline-definition::get-max-poll-timeout"
	if d.MaxPollTimeoutMs != "" {
		if tm, err := strconv.Atoi(d.MaxPollTimeoutMs); err == nil {
			return tm
		} else {
			log.Error().Str("max-poll-timeout-str", d.MaxPollTimeoutMs).Msg(semLogContext)
		}
	}

	return d.MaxPollTimeout
}

func (d *RhpPipelineDefinition) Enabled() bool {
	if d.En == "" || d.En == "true" {
		return true
	}

	return false
}

func (d *RhpPipelineDefinition) ErrorPolicyForError(err error) string {
	return tprod.ErrorPolicyForError(err, d.OnErrors)
}

func (d *RhpPipelineDefinition) NumPartitionsAsInt() int {
	const semLogContext = "pipeline-definition::num-partitions"

	if d.NumPartitions == "" {
		log.Info().Msg(semLogContext + " - num-partitions not set: setting to 1")
		return 1
	}
	n, err := strconv.Atoi(d.NumPartitions)
	if err != nil || n <= 0 {
		log.Error().Err(err).Str("num-partitions", d.NumPartitions).Msg(semLogContext)
		n = 1
	}

	return n
}

func DeserializeRhpPipelineDefinitionFromYAMLFile(fn string) (RhpPipelineDefinition, error) {

	const semLogContext = "pipeline-definition::deserialize-from-yaml-file"
	pl := RhpPipelineDefinition{}

	b, err := util.ReadFileAndResolveEnvVars(fn)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return pl, err
	}

	log.Info().Str("pl-cfg", string(b)).Msg(semLogContext)

	err = yaml.Unmarshal(b, &pl)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	if len(pl.OnErrors) == 0 {
		log.Warn().Msg(semLogContext + " - please set OnErrors property using exit for all levels")
		pl.OnErrors = []tprod.OnErrorPolicy{
			{ErrLevel: tprod.OnErrorLevelSystem, Policy: tprod.OnErrorExit},
			{ErrLevel: tprod.OnErrorLevelFatal, Policy: tprod.OnErrorExit},
			{ErrLevel: tprod.OnErrorLevelError, Policy: tprod.OnErrorExit},
		}
	}

	log.Info().Int("num-partitions", pl.NumPartitionsAsInt()).Msg(semLogContext)
	return pl, err
}

func (d *RhpPipelineDefinition) WriteToFolder(folderName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "pipeline-definition::write-to-file"
	fn := filepath.Join(folderName, RhpPipelineDefinitionFileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(d)
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
