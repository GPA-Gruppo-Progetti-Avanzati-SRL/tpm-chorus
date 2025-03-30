package kafkasink

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type StatsInfo struct {
	NumQueuedMessages   uint64
	NumProducedMessages uint64
	NumAcceptedMessages uint64
	NumFailedMessages   uint64

	Status200Metric promutil.CollectorWithLabels
	Status202Metric promutil.CollectorWithLabels
	Status500Metric promutil.CollectorWithLabels
	metricErrors    bool
}

func (stat *StatsInfo) Clear() *StatsInfo {
	stat.NumQueuedMessages = 0
	stat.NumProducedMessages = 0
	stat.NumAcceptedMessages = 0
	stat.NumFailedMessages = 0
	return stat
}

const (
	MetricLabelName       = "name"
	MetricLabelStatusCode = "status-code"
	MetricLabelTopicName  = "topic-name"
)

func NewStatsInfo(brokerName, topicName, metricGroupId, metricId string) *StatsInfo {
	stat := &StatsInfo{}
	mg, err := promutil.GetGroup(metricGroupId)
	if err != nil {
		stat.metricErrors = true
		return stat
	} else {
		stat.Status200Metric, err = mg.CollectorByIdWithLabels(metricId, map[string]string{
			MetricLabelName:       brokerName,
			MetricLabelStatusCode: "200",
			MetricLabelTopicName:  "topic-name",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.Status202Metric, err = mg.CollectorByIdWithLabels(metricId, map[string]string{
			MetricLabelName:       brokerName,
			MetricLabelStatusCode: "202",
			MetricLabelTopicName:  "topic-name",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.Status500Metric, err = mg.CollectorByIdWithLabels(metricId, map[string]string{
			MetricLabelName:       brokerName,
			MetricLabelStatusCode: "500",
			MetricLabelTopicName:  "topic-name",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}
	}

	return stat
}

func (stat *StatsInfo) IsZero() bool {
	return stat.NumQueuedMessages == 0 && stat.NumProducedMessages == 0 && stat.NumAcceptedMessages == 0 && stat.NumFailedMessages == 0
}

func (stat *StatsInfo) IsFail() bool {
	return stat.NumFailedMessages > 0
}

func (stat *StatsInfo) IsComplete() bool {
	return stat.NumQueuedMessages == (stat.NumProducedMessages + stat.NumFailedMessages)
}

func (stat *StatsInfo) Log(semLogContext string, expectedCompleted bool) {
	var err error
	var evt *zerolog.Event
	if stat.IsFail() {
		err = errors.New("kafka stage with some failures")
		evt = log.Error().Err(err)
	} else if !stat.IsComplete() && expectedCompleted {
		err = errors.New("kafka stage not completed, but should be")
		evt = log.Error().Err(err)
	} else {
		evt = log.Trace()
	}

	evt.Uint64("num-failed", stat.NumFailedMessages).
		Uint64("num-accepted", stat.NumAcceptedMessages).
		Uint64("num-produced", stat.NumProducedMessages).
		Uint64("num-queued", stat.NumQueuedMessages).Msg(semLogContext)
}

func (stat *StatsInfo) IsErr(expectedCompleted bool) error {
	var err error
	if stat.IsFail() {
		err = errors.New("kafka stage with some failures")
	} else if !stat.IsComplete() && expectedCompleted {
		err = errors.New("kafka stage not completed, but should be")
	}

	return err
}

func (stat *StatsInfo) ProduceMetrics() {
	if !stat.metricErrors {
		stat.Status202Metric.SetMetric(float64(stat.NumAcceptedMessages))
		stat.Status200Metric.SetMetric(float64(stat.NumProducedMessages))
		stat.Status500Metric.SetMetric(float64(stat.NumFailedMessages))
	}
}
