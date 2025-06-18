package tbmconfig

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plconfig/sinkconfig"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"

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
	TubaMirumPipelineDefinitionFileName = "tpm-tuba-mirum-pipeline.yml"
)

type TbmPipelineEventDefinition struct {
	ContentType string `json:"content-type,omitempty" yaml:"content-type,omitempty" mapstructure:"content-type,omitempty"`
	Schema      string `json:"schema,omitempty" yaml:"schema,omitempty" mapstructure:"schema,omitempty"`
}

func (p *TbmPipelineEventDefinition) IsZero() bool {
	if p.ContentType == "" && p.Schema == "" {
		return true
	}

	return false
}

type TbmPipelineDestination struct {
	SingStage string `json:"sink-stage,omitempty" yaml:"sink-stage,omitempty" mapstructure:"sink-stage,omitempty"`
	Guard     string `json:"guard,omitempty" yaml:"guard,omitempty" mapstructure:"guard,omitempty"`
}

type PathDefinition struct {
	OrchestrationFolder string                     `json:"orchestration-folder,omitempty" yaml:"orchestration-folder,omitempty" mapstructure:"orchestration-folder,omitempty"`
	EventInfo           TbmPipelineEventDefinition `json:"event,omitempty" yaml:"event,omitempty" mapstructure:"event,omitempty"`
	Destinations        []TbmPipelineDestination   `json:"destinations,omitempty" yaml:"destinations,omitempty" mapstructure:"destinations,omitempty"`
}

func (p *PathDefinition) IsZero() bool {
	if p.OrchestrationFolder == "" && len(p.Destinations) == 0 && p.EventInfo.IsZero() {
		return true
	}

	return false
}

// OnErrors            []OnErrorPolicy                     `yaml:"on-errors,omitempty" mapstructure:"on-errors,omitempty" json:"on-errors,omitempty"`

type PipelineDefinition struct {
	Id                         string                                    `json:"id,omitempty" yaml:"id,omitempty" mapstructure:"id,omitempty"`
	En                         string                                    `json:"enabled,omitempty" yaml:"enabled,omitempty" mapstructure:"enabled,omitempty"`
	Description                string                                    `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`
	WorkMode                   string                                    `yaml:"work-mode,omitempty" mapstructure:"work-mode,omitempty" json:"work-mode,omitempty"`
	EventJsonSerializationMode string                                    `yaml:"event-json-ser-mode,omitempty" mapstructure:"event-json-ser-mode,omitempty" json:"event-json-ser-mode,omitempty"`
	NumPartitions              string                                    `yaml:"num-partitions,omitempty" mapstructure:"num-partitions,omitempty" json:"num-partitions,omitempty"`
	TickInterval               string                                    `yaml:"tick-interval,omitempty" mapstructure:"tick-interval,omitempty" json:"tick-interval,omitempty"`
	MaxBatchSize               int                                       `yaml:"max-batch-size,omitempty" mapstructure:"max-batch-size,omitempty" json:"max-batch-size,omitempty"`
	RefMetrics                 *promutil.MetricsConfigReference          `yaml:"ref-metrics"  mapstructure:"ref-metrics"  json:"ref-metrics"`
	SpanName                   string                                    `yaml:"tracing-span-name,omitempty" mapstructure:"tracing-span-name,omitempty" json:"tracing-span-name,omitempty"`
	DeadLetterTopic            string                                    `json:"dead-letter-topic,omitempty" yaml:"dead-letter-topic,omitempty" mapstructure:"dead-letter-topic,omitempty"`
	Paths                      []PathDefinition                          `json:"paths,omitempty" yaml:"paths,omitempty" mapstructure:"paths,omitempty"`
	Sinks                      []sinkconfig.SinkStageDefinitionReference `json:"sink-stages,omitempty" yaml:"sink-stages,omitempty" mapstructure:"sink-stages,omitempty"`
}

func (d *PipelineDefinition) Enabled() bool {
	if d.En == "" || d.En == "true" {
		return true
	}

	return false
}

// func (d *Definition) ErrorPolicyForError(err error) string {
//	return ErrorPolicyForError(err, d.OnErrors)
//}

func (d *PipelineDefinition) NumPartitionsAsInt() int {
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

func DeserializePipelineFromYAMLFile(fn string) (PipelineDefinition, error) {

	const semLogContext = "pipeline-definition::deserialize-from-yaml-file"
	pl := PipelineDefinition{}

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

	log.Info().Int("num-partitions", pl.NumPartitionsAsInt()).Msg(semLogContext)
	return pl, err
}

func (d *PipelineDefinition) WriteToFolder(folderName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "pipeline-definition::write-to-file"
	fn := filepath.Join(folderName, TubaMirumPipelineDefinitionFileName)
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

type TbmPipelineOnErrorPolicy struct {
	ErrLevel string `yaml:"level,omitempty" mapstructure:"level,omitempty" json:"level,omitempty"`
	Policy   string `yaml:"policy,omitempty" mapstructure:"policy,omitempty" json:"policy,omitempty"`
}

func TbmPipelineErrorPolicyForError(err error, onErrors []TbmPipelineOnErrorPolicy) string {

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
