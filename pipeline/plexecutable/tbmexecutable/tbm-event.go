package tbmexecutable

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable/orchestration"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jobs/taskconsumer/datasource"
)

type TbmEvent struct {
	Event             *datasource.Event
	BatchPosition     int
	BatchPartition    int
	WfCase            *wfcase.WfCase `yaml:"-" mapstructure:"-" json:"-"`
	SelectedPathIndex int
	Orchestration     *orchestration.Orchestration `yaml:"-" mapstructure:"-" json:"-"`
}
