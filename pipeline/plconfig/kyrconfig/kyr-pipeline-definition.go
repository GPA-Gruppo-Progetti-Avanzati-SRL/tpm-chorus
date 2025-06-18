package kyrconfig

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plconfig/sinkconfig"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/changestream/checkpoint/factory"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/changestream/consumerproducer"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
)

const (
	OnErrorLevelFatal  = "fatal"
	OnErrorLevelSystem = "system"
	OnErrorLevelError  = "error"

	OnErrorExit       = "exit"
	OnErrorDeadLetter = "dead-letter"

	OnEofExit = "exit"

	WorkModeMsg            = "msg-mode"
	WorkModeBatch          = "batch-mode"
	WorkModeBatchFF        = "batch-mode-ff"
	EventJsonModeCanonical = "canonical"
	EventJsonModeRelaxed   = "relaxed"
)

const (
	KyrPipelineDefinitionFileName = "tpm-kyrie-pipeline.yml"
)

type KyrPipelineEventDefinition struct {
	ContentType string `json:"content-type,omitempty" yaml:"content-type,omitempty" mapstructure:"content-type,omitempty"`
	Schema      string `json:"schema,omitempty" yaml:"schema,omitempty" mapstructure:"schema,omitempty"`
}

func (p *KyrPipelineEventDefinition) IsZero() bool {
	if p.ContentType == "" && p.Schema == "" {
		return true
	}

	return false
}

type KyrPipelineDestination struct {
	SingStage string `json:"sink-stage,omitempty" yaml:"sink-stage,omitempty" mapstructure:"sink-stage,omitempty"`
	Guard     string `json:"guard,omitempty" yaml:"guard,omitempty" mapstructure:"guard,omitempty"`
}

type KyrPipelinePathDefinition struct {
	OrchestrationFolder string                     `json:"orchestration-folder,omitempty" yaml:"orchestration-folder,omitempty" mapstructure:"orchestration-folder,omitempty"`
	EventInfo           KyrPipelineEventDefinition `json:"event,omitempty" yaml:"event,omitempty" mapstructure:"event,omitempty"`
	Destinations        []KyrPipelineDestination   `json:"destinations,omitempty" yaml:"destinations,omitempty" mapstructure:"destinations,omitempty"`
}

func (p *KyrPipelinePathDefinition) IsZero() bool {
	if p.OrchestrationFolder == "" && len(p.Destinations) == 0 && p.EventInfo.IsZero() {
		return true
	}

	return false
}

// OnErrors            []OnErrorPolicy                     `yaml:"on-errors,omitempty" mapstructure:"on-errors,omitempty" json:"on-errors,omitempty"`

type KyrPipelineDefinition struct {
	Id                         string                                    `json:"id,omitempty" yaml:"id,omitempty" mapstructure:"id,omitempty"`
	En                         string                                    `json:"enabled,omitempty" yaml:"enabled,omitempty" mapstructure:"enabled,omitempty"`
	Description                string                                    `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`
	WorkMode                   string                                    `yaml:"work-mode,omitempty" mapstructure:"work-mode,omitempty" json:"work-mode,omitempty"`
	WithChannel                bool                                      `yaml:"with-channel,omitempty" mapstructure:"with-channel,omitempty" json:"with-channel,omitempty"`
	WithChannelSize            int                                       `yaml:"with-channel-size,omitempty" mapstructure:"with-channel-size,omitempty" json:"with-channel-size,omitempty"`
	EventJsonSerializationMode string                                    `yaml:"event-json-ser-mode,omitempty" mapstructure:"event-json-ser-mode,omitempty" json:"event-json-ser-mode,omitempty"`
	NumPartitions              string                                    `yaml:"num-partitions,omitempty" mapstructure:"num-partitions,omitempty" json:"num-partitions,omitempty"`
	TickInterval               string                                    `yaml:"tick-interval,omitempty" mapstructure:"tick-interval,omitempty" json:"tick-interval,omitempty"`
	MaxBatchSize               int                                       `yaml:"max-batch-size,omitempty" mapstructure:"max-batch-size,omitempty" json:"max-batch-size,omitempty"`
	RefMetrics                 *promutil.MetricsConfigReference          `yaml:"ref-metrics"  mapstructure:"ref-metrics"  json:"ref-metrics"`
	SpanName                   string                                    `yaml:"tracing-span-name,omitempty" mapstructure:"tracing-span-name,omitempty" json:"tracing-span-name,omitempty"`
	DeadLetterTopic            string                                    `json:"dead-letter-topic,omitempty" yaml:"dead-letter-topic,omitempty" mapstructure:"dead-letter-topic,omitempty"`
	Paths                      []KyrPipelinePathDefinition               `json:"paths,omitempty" yaml:"paths,omitempty" mapstructure:"paths,omitempty"`
	Sinks                      []sinkconfig.SinkStageDefinitionReference `json:"sink-stages,omitempty" yaml:"sink-stages,omitempty" mapstructure:"sink-stages,omitempty"`
	Consumer                   consumerproducer.ConsumerConfig           `yaml:"consumer,omitempty" mapstructure:"consumer,omitempty" json:"consumer,omitempty"`
	CheckPointSvcConfig        factory.Config                            `yaml:"checkpoint-svc,omitempty" mapstructure:"checkpoint-svc,omitempty" json:"checkpoint-svc,omitempty"`
}

func (d *KyrPipelineDefinition) Enabled() bool {
	if d.En == "" || d.En == "true" {
		return true
	}

	return false
}

// func (d *Definition) ErrorPolicyForError(err error) string {
//	return ErrorPolicyForError(err, d.OnErrors)
//}

func (d *KyrPipelineDefinition) NumPartitionsAsInt() int {
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

func DeserializeKyrPipelineFromYAMLFile(fn string) (KyrPipelineDefinition, error) {

	const semLogContext = "pipeline-definition::deserialize-from-yaml-file"
	pl := KyrPipelineDefinition{}

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

	if pl.Consumer.OnErrorPolicy != consumerproducer.OnErrorExit && pl.Consumer.OnErrorPolicy != consumerproducer.OnErrorRewind {
		log.Warn().Msg(semLogContext + " - please explicitly set on-error to exit or rewind (using exit as default")
		pl.Consumer.OnErrorPolicy = consumerproducer.OnErrorExit
	}

	if pl.WithChannel && pl.WithChannelSize <= 0 {
		pl.WithChannelSize = 2
	}

	/*
		if len(pl.OnErrors) == 0 {
			log.Warn().Msg(semLogContext + " - please set OnErrors property using exit for all levels")
			pl.OnErrors = []OnErrorPolicy{
				{ErrLevel: OnErrorLevelSystem, Policy: OnErrorExit},
				{ErrLevel: OnErrorLevelFatal, Policy: OnErrorExit},
				{ErrLevel: OnErrorLevelError, Policy: OnErrorExit},
			}
		}
	*/

	log.Info().Int("num-partitions", pl.NumPartitionsAsInt()).Msg(semLogContext)
	if pl.isPipelineEligible4RawProcessing() {
		pl.WorkMode = WorkModeBatchFF
		log.Warn().Str("pipeline-mode", pl.WorkMode).Msg(semLogContext)
	}

	return pl, err
}

func (d *KyrPipelineDefinition) isPipelineEligible4RawProcessing() bool {
	const semLogContext = "pipeline-definition::is-pipeline-eligible-4-bulk-processing"

	if d.WorkMode != WorkModeBatch {
		log.Info().Msg(semLogContext + " - for this scenario work-mode should be set to " + WorkModeBatch)
		return false
	}

	if len(d.Paths) > 1 {
		log.Info().Msg(semLogContext + " - only one path is supported for raw scenario")
		return false
	}

	for _, path := range d.Paths {
		if path.OrchestrationFolder != "" {
			log.Info().Msg(semLogContext + " - orchestrations are not supported for raw scenario")
			return false
		}

		if len(path.Destinations) > 1 {
			log.Info().Msg(semLogContext + " - the single path scenario admits at maximum only one destination")
			return false
		}

		for _, dest := range path.Destinations {
			if dest.Guard != "" && dest.Guard != "true" {
				log.Info().Msg(semLogContext + " - the single destination has to be const enabled")
				return false
			}
		}
	}

	if len(d.Sinks) > 1 {
		log.Info().Msg(semLogContext + " - only one sink is supported for raw scenario")
		return false
	}

	for _, sink := range d.Sinks {
		if sink.Typ != sinkconfig.SinkTypeKafkaFF {
			log.Info().Msg(semLogContext + " - only kafka-ff sink is supported for raw scenario")
			return false
		}
	}

	log.Info().Msg(semLogContext + " - pipeline is eligible for batch raw processing")
	return true
}

func (d *KyrPipelineDefinition) WriteToFolder(folderName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "pipeline-definition::write-to-file"
	fn := filepath.Join(folderName, KyrPipelineDefinitionFileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(d)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	err = fileutil.WriteFile(fn, b, os.ModePerm, writeOpts...)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

type KyrPipelineOnErrorPolicy struct {
	ErrLevel string `yaml:"level,omitempty" mapstructure:"level,omitempty" json:"level,omitempty"`
	Policy   string `yaml:"policy,omitempty" mapstructure:"policy,omitempty" json:"policy,omitempty"`
}

func KyrPipelineErrorPolicyForError(err error, onErrors []KyrPipelineOnErrorPolicy) string {

	level := OnErrorLevelFatal
	//var tprodErr *TransformerProducerError
	//if errors.As(err, &tprodErr) {
	//	level = tprodErr.Level
	//}

	foundExit := false
	for _, c := range onErrors {
		if c.ErrLevel == level {
			return c.Policy
		}

		if c.Policy == OnErrorExit {
			foundExit = true
		}
	}

	if foundExit {
		return OnErrorExit
	}

	return OnErrorDeadLetter
}
