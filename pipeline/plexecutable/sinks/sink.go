package sinks

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plexecutable"
)

type Event struct {
	Primitive      interface{}
	WfCase         *wfcase.WfCase
	BatchPosition  int
	BatchPartition int
	ResumeToken    string
}

type Stage interface {
	Id() string
	Type() string
	Sink(evt *plexecutable.PipelineEvent) error
	Flush() (int, error)
	Clear() int
	Close()
	Reset() error
}
