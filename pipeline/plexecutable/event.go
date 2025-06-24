package plexecutable

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
)

const (
	PipelineEventPrimitive         = "primitive"
	PipelineEventBatchPosition     = "batchposition"
	PipelineEventBatchPartition    = "batchpartition"
	PipelineEventResumeToken       = "resume-token"
	PipelineEventWfCase            = "wfcase"
	PipelineEventSelectedPathIndex = "path-index"
	PipelineEVentOrchestration     = "orchestration"
)

type PipelineEvent struct {
	Primitive         interface{} `yaml:"-" mapstructure:"-" json:"-"`
	BatchPosition     int
	BatchPartition    int
	WfCase            *wfcase.WfCase `yaml:"-" mapstructure:"-" json:"-"`
	SelectedPathIndex int
	Orchestration     *orchestration.Orchestration `yaml:"-" mapstructure:"-" json:"-"`
	ResumeToken       string
}
