package executable

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/changestream/events"
)

type KyrEvent struct {
	ChangeEvent       *events.ChangeEvent
	BatchPosition     int
	BatchPartition    int
	WfCase            *wfcase.WfCase `yaml:"-" mapstructure:"-" json:"-"`
	SelectedPathIndex int
	Orchestration     *orchestration.Orchestration `yaml:"-" mapstructure:"-" json:"-"`
}
