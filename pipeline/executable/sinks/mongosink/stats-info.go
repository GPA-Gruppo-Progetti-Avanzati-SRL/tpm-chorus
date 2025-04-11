package mongosink

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type StatsInfo struct {
	ModifiedCount  int64
	InsertedCount  int64
	UpsertedCount  int64
	MatchedCount   int64
	DeletedCount   int64
	ErrsCount      int64
	DiscardedCount int64
	BulkSize       int

	WriteModifiedCountMetric promutil.CollectorWithLabels
	WriteInsertedCountMetric promutil.CollectorWithLabels
	WriteUpsertedCountMetric promutil.CollectorWithLabels
	WriteMatchedCountMetric  promutil.CollectorWithLabels
	WriteDeletedCountMetric  promutil.CollectorWithLabels
	WriteErrsCountMetric     promutil.CollectorWithLabels
	WriteBulkSizeGaugeMetric promutil.CollectorWithLabels
	SinkDiscardedCountMetric promutil.CollectorWithLabels
	WriteDurationHistMetric  promutil.CollectorWithLabels
	metricErrors             bool
}

func (stat *StatsInfo) Clear() *StatsInfo {
	stat.ModifiedCount = 0
	stat.InsertedCount = 0
	stat.UpsertedCount = 0
	stat.MatchedCount = 0
	stat.DeletedCount = 0
	stat.ErrsCount = 0
	stat.BulkSize = 0
	stat.DiscardedCount = 0
	return stat
}

const (
	WriteDurationHistogramMetric = "mdb-write-duration"
	WriteBulkOpsMetric           = "mdb-writes-ops"
	WriteErrsMetric              = "mdb-write-errors"
	WriteBulkSizeGaugeMetric     = "mdb-write-bulk-size"
	SinkDiscardedCountMetric     = "mdb-sink-discarded-count"
	MetricLabelStageId           = "name"
	MetricLabelPipelineId        = "pipeline-id"
	MetricLabelOpCounter         = "oper"
)

func NewStatsInfo(pipelineId, stageId, collectionName string, metricGroupId string) *StatsInfo {
	const semLogContext = "mongo-sink::mew-stats-info"

	stat := &StatsInfo{}
	mg, err := promutil.GetGroup(metricGroupId)
	if err != nil {
		stat.metricErrors = true
		log.Error().Err(err).Str("stageId", stageId).Msg(semLogContext)
		return stat
	} else {

		stat.WriteInsertedCountMetric, err = mg.CollectorByIdWithLabels(WriteBulkOpsMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "inserted",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteUpsertedCountMetric, err = mg.CollectorByIdWithLabels(WriteBulkOpsMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "upserted",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteModifiedCountMetric, err = mg.CollectorByIdWithLabels(WriteBulkOpsMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "modified",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteMatchedCountMetric, err = mg.CollectorByIdWithLabels(WriteBulkOpsMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "matched",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteDeletedCountMetric, err = mg.CollectorByIdWithLabels(WriteBulkOpsMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "deleted",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteErrsCountMetric, err = mg.CollectorByIdWithLabels(WriteErrsMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "errors",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteBulkSizeGaugeMetric, err = mg.CollectorByIdWithLabels(WriteBulkSizeGaugeMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
			MetricLabelOpCounter:  "errors",
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.SinkDiscardedCountMetric, err = mg.CollectorByIdWithLabels(SinkDiscardedCountMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}

		stat.WriteDurationHistMetric, err = mg.CollectorByIdWithLabels(WriteDurationHistogramMetric, map[string]string{
			MetricLabelPipelineId: pipelineId,
			MetricLabelStageId:    stageId,
		})
		if err != nil {
			stat.metricErrors = true
			return stat
		}
	}

	return stat
}

func (stat *StatsInfo) SetBulkSize(bulkSize int) {

	stat.BulkSize = bulkSize
	if !stat.metricErrors {
		stat.WriteBulkSizeGaugeMetric.SetMetric(float64(bulkSize))
	}
}

func (stat *StatsInfo) IncDiscarded(n int64) {
	stat.DiscardedCount = n
	if !stat.metricErrors {
		stat.SinkDiscardedCountMetric.SetMetric(float64(n))
	}
}

func (stat *StatsInfo) IncErrors(errs int64) {

	stat.ErrsCount += errs
	if !stat.metricErrors {
		stat.WriteErrsCountMetric.SetMetric(float64(errs))
	}
}

func (stat *StatsInfo) Update(result *mongo.BulkWriteResult, writeDuration time.Duration) {
	stat.DeletedCount += result.DeletedCount
	stat.InsertedCount += result.InsertedCount
	stat.MatchedCount += result.MatchedCount
	stat.ModifiedCount += result.ModifiedCount
	stat.UpsertedCount += result.UpsertedCount

	if !stat.metricErrors {
		stat.WriteMatchedCountMetric.SetMetric(float64(result.MatchedCount))
		stat.WriteModifiedCountMetric.SetMetric(float64(result.ModifiedCount))
		stat.WriteInsertedCountMetric.SetMetric(float64(result.InsertedCount))
		stat.WriteUpsertedCountMetric.SetMetric(float64(result.UpsertedCount))
		stat.WriteDeletedCountMetric.SetMetric(float64(result.DeletedCount))
		stat.WriteDurationHistMetric.SetMetric(writeDuration.Seconds())
	}
}
