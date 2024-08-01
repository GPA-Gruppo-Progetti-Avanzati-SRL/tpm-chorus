package reporter

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
)

const (
	DefaultMetricsEmbeddedGroupId = "local"
	DefaultMetricsGroupId         = "har-reporting"
	DefaultCounterId              = "har-messages"
)

/*
type MetricsCfg struct {
	GId       string `yaml:"group-id,omitempty" mapstructure:"group-id,omitempty" json:"group-id,omitempty"`
	CounterId string `yaml:"counter-id,omitempty" mapstructure:"counter-id,omitempty" json:"counter-id,omitempty"`
}

func (mCfg MetricsCfg) IsEnabled() bool {
	return mCfg.GId != "-"
}

func (mCfg MetricsCfg) IsCounterEnabled() bool {
	return mCfg.GId != "-" && mCfg.CounterId != "-"
}
*/

var DefaultMetricsCfg = promutil.MetricsConfigReference{
	GId:       DefaultMetricsGroupId,
	CounterId: DefaultCounterId,
}

type Config struct {
	ReporterType     string                           `mapstructure:"type" yaml:"type" json:"type"`
	QueueSize        int                              `mapstructure:"work-queue" yaml:"work-queue" json:"work-queue"`
	DetailLevel      wfcase.ReportLogDetail           `mapstructure:"level" yaml:"level" json:"level"`
	PiiFile          string                           `mapstructure:"pii-config-fn" yaml:"pii-config-fn" json:"pii-config-fn"`
	Dummy            *DummyReporterConfig             `mapstructure:"dummy" yaml:"dummy" json:"dummy"`
	Kafka            *KafkaConfig                     `mapstructure:"kafka" yaml:"kafka" json:"kafka"`
	RefMetricsConfig *promutil.MetricsConfigReference `mapstructure:"ref-metrics,omitempty" yaml:"ref-metrics,omitempty" json:"ref-metrics,omitempty"`
	MetricsConfig    promutil.MetricGroupConfig       `mapstructure:"metrics" yaml:"metrics" json:"metrics"`
}
