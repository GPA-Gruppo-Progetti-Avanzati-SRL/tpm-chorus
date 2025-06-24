package kafkasink

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/tprod"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"net/http"
	"sync"
	"time"
)

const MetricMessagesToTopic = "cdc-events-to-topic"

type QueueMessage struct {
	BatchPosition  int
	BatchPartition int
	Rt             interface{}
	Seq            int
	Span           opentracing.Span
	ToTopic        string
	Headers        map[string]string
	Key            []byte
	Body           []byte
}

//type KafkaSinkStageMessageQueueFlushConfig struct {
//	FlushTimeoutMs             int `yaml:"prd-flush-timeout-ms,omitempty"  json:"prd-flush-timeout-ms,omitempty" mapstructure:"prd-flush-timeout-ms,omitempty"`
//	FlushStride                int `yaml:"prd-flush-stride,omitempty"  json:"prd-flush-stride,omitempty" mapstructure:"prd-flush-stride,omitempty"`
//	OnCompletionFlushTimeoutMs int `yaml:"onc-flush-timeout-ms,omitempty"  json:"onc-flush-timeout-ms,omitempty" mapstructure:"onc-flush-timeout-ms,omitempty"`
//}
//
//func (cfg KafkaSinkStageMessageQueueFlushConfig) FlushWhenProducingToTopics(messageNum int) bool {
//	b := cfg.FlushTimeoutMs != 0 && cfg.FlushStride != 0 && messageNum%cfg.FlushStride == 0
//	return b
//}
//
//func (cfg KafkaSinkStageMessageQueueFlushConfig) FlushOnCompletion() bool {
//	return cfg.OnCompletionFlushTimeoutMs != 0
//}

type KafkaSinkStageBufferedMessageQueue struct {
	PipelineWorkMode string

	BufferSize      int
	FlushTimeout    time.Duration `yaml:"flush-timeout-ms,omitempty"  json:"flush-timeout-ms,omitempty" mapstructure:"flush-timeout-ms,omitempty"`
	statsInfo       *StatsInfo
	WithRandomError util.ErrorRandomizer
	produceSequence int

	Kp             *tprod.KafkaProducerWrapper
	Items          []QueueMessage
	MetricsGroupId string
	MetricsLabels  map[string]string
	mu             sync.Mutex
}

func NewKafkaSinkStageBufferedMessageQueue(pipelineWorkMode, brokerName string, kp *tprod.KafkaProducerWrapper, bufSize int, flushTimeout time.Duration, metricGroupId string, withRandomError string) (*KafkaSinkStageBufferedMessageQueue, error) {
	const semLogContext = "kafka-sink-stage-buffered-queue::new"

	var err error
	var rnd util.ErrorRandomizer
	rnd, err = util.NewErrorRandomizer(withRandomError)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	mq := &KafkaSinkStageBufferedMessageQueue{
		PipelineWorkMode: pipelineWorkMode,
		BufferSize:       bufSize,
		FlushTimeout:     flushTimeout,
		MetricsGroupId:   metricGroupId,
		Kp:               kp,
		statsInfo:        NewStatsInfo(brokerName, "my-topic", metricGroupId, MetricMessagesToTopic),
		WithRandomError:  rnd,
	}

	return mq, nil
}

func (q *KafkaSinkStageBufferedMessageQueue) SetKafkaProducerWrapper(kp *tprod.KafkaProducerWrapper) {
	q.Kp = kp
}

//func (q *KafkaSinkStageBufferedMessageQueue) DeliveryInfo() DeliveryStats {
//	return q.DeliveryStats
//}

func (q *KafkaSinkStageBufferedMessageQueue) IsRandomError() error {
	const semLogContext = "kafka-sink-stage-buffered-queue::is-random-error"
	if !util.IsNilish(q.WithRandomError) {
		return q.WithRandomError.GenerateRandomError()
	}

	return nil
}

func (q *KafkaSinkStageBufferedMessageQueue) isBuffered() bool {
	return q.BufferSize > 1
}

func (q *KafkaSinkStageBufferedMessageQueue) isOverQuota() bool {
	return len(q.Items) >= q.BufferSize
}

func (stage *KafkaSinkStageBufferedMessageQueue) Flush( /*waitCfg plconfig.KafkaSinkOnNotCompleted*/ ) (int, error) {
	const semLogContext = "kafka-sink-stage-buffered-queue::flush"

	sz := len(stage.Items)
	err := stage.flushQueue()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return sz, err
	}

	flushTimeoutMs := util.IntCoalesce(int(stage.FlushTimeout.Milliseconds()), int((10 * time.Second).Milliseconds()))
	numberOfUnFlushedEvents := stage.Kp.Flush(flushTimeoutMs)
	if numberOfUnFlushedEvents > 0 {
		err = errors.New("sink-stage flush timeout")
		log.Error().Err(err).Int("number-of-un-flushed-events", numberOfUnFlushedEvents).Int64("timeout-ms", stage.FlushTimeout.Milliseconds()).Msg(semLogContext)
	}

	if err == nil && !util.IsNilish(stage.WithRandomError) {
		err = stage.IsRandomError()
	}

	stage.statsInfo.ProduceMetrics()
	errStats := stage.statsInfo.IsErr(true)
	if errStats != nil {
		stage.statsInfo.Log(semLogContext, true)
	}
	stage.statsInfo.Clear()

	return sz, err
}

func (q *KafkaSinkStageBufferedMessageQueue) Clear() {
	// Era And ma forse doveva essere Add.... e cmq forse non ha senso sottrarlo... in quanto bisogna capire cosa sottraggo..... perche' potrebbero essere gia' contabilizzati nei produced e failed.
	// Per il momento lo rimetto e tolgo il Reset Viene invocato quando ho un errore di orchestrazione direi e quindi se e' bufferizzato quelli che ho in canna dovrebbero essere ancora da inviare.
	//q.numQueuedMessages.Add(uint64(-len(q.Items)))
	q.statsInfo.Clear()
	// q.Reset()
}

func (p *KafkaSinkStageBufferedMessageQueue) Produce(msgs ...QueueMessage) error {
	const semLogContext = "kafka-sink-stage-buffered-queue::produce"
	var err error

	p.statsInfo.NumQueuedMessages += uint64(len(msgs))
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isBuffered() {
		err = p.produce2Topics(p.Kp, msgs)
		if err != nil {
			tprod.LogKafkaError(err).Msg(semLogContext)
		}
	} else {
		p.Items = append(p.Items, msgs...)
		if p.isOverQuota() {
			err = p.produce2Topics(p.Kp, p.Items)
			if err != nil {
				tprod.LogKafkaError(err).Msg(semLogContext)
			}
			p.Items = nil
		}
	}

	if err != nil {
		tprod.LogKafkaError(err).Msg(semLogContext)
	}

	return err
}

func (p *KafkaSinkStageBufferedMessageQueue) Close() error {
	const semLogContext = "kafka-sink-stage-buffered-queue::close"
	err := p.flushQueue()
	p.statsInfo.Clear()
	p.Items = nil
	p.Kp.Close()
	return err
}

func (p *KafkaSinkStageBufferedMessageQueue) flushQueue() error {
	const semLogContext = "kafka-sink-stage-buffered-queue::flush-queue"
	var err error

	if len(p.Items) > 0 {
		err = p.produce2Topics(p.Kp, p.Items)
		if err != nil {
			tprod.LogKafkaError(err).Msg(semLogContext)
		}
	}

	p.Items = nil
	return err
}

func (p *KafkaSinkStageBufferedMessageQueue) produce2Topics(kp *tprod.KafkaProducerWrapper, messages []QueueMessage) error {
	const semLogContext = "kafka-sink-stage-buffered-queue::produce-to-topics"

	for _, m := range messages {

		// Producing a random error
		if err := p.IsRandomError(); err != nil {
			return err
		}

		if _, err := p.produce2Topic(kp, m); err != nil {
			return err
		}
	}

	return nil
}

func (p *KafkaSinkStageBufferedMessageQueue) produce2Topic(kp *tprod.KafkaProducerWrapper, m QueueMessage) (int, error) {
	const semLogContext = "kafka-sink-stage-buffered-queue::produce-to-topic"

	var hdrs []kafka.Header
	for n, h := range m.Headers {
		hdrs = append(hdrs, kafka.Header{Key: n, Value: []byte(h)})
	}
	km := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &m.ToTopic, Partition: kafka.PartitionAny},
		Key:            m.Key,
		Value:          m.Body,
		Headers:        hdrs,
	}

	if m.Span != nil {

		headers := make(map[string]string)
		opentracing.GlobalTracer().Inject(
			m.Span.Context(),
			opentracing.TextMap,
			opentracing.TextMapCarrier(headers))

		for headerKey, headerValue := range headers {
			km.Headers = append(km.Headers, kafka.Header{
				Key:   headerKey,
				Value: []byte(headerValue),
			})
		}
	}

	st, err := kp.Produce(km)
	if err != nil {
		log.Err(err).Msg(semLogContext)
	}

	switch st {
	case http.StatusOK:
		p.statsInfo.NumProducedMessages++
		p.statsInfo.NumAcceptedMessages++
	case http.StatusAccepted:
		p.statsInfo.NumAcceptedMessages++
	default:
		p.statsInfo.NumFailedMessages++
	}

	return st, err
}

func (p *KafkaSinkStageBufferedMessageQueue) MonitorProducerEvents(name string) {
	const semLogContext = "kafka-sink-stage-buffered-queue::monitor-producer"
	log.Info().Msg(semLogContext + " starting monitor producer events")

	exitFromLoop := false

	for e := range p.Kp.Events() {

		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				logEvt := tprod.LogKafkaError(ev.TopicPartition.Error).Int64("offset", int64(ev.TopicPartition.Offset)).Int32("partition", ev.TopicPartition.Partition).Interface("topic", ev.TopicPartition.Topic)
				logEvt.Msg(semLogContext + " delivery failed")
				p.statsInfo.NumFailedMessages++

			} else {
				logEvt := log.Trace().Int64("offset", int64(ev.TopicPartition.Offset)).Int32("partition", ev.TopicPartition.Partition).Interface("topic", ev.TopicPartition.Topic)
				logEvt.Msg(semLogContext + " delivery ok")
				p.statsInfo.NumProducedMessages++
			}
		}

		if exitFromLoop {
			break
		}
	}

	log.Info().Msg(semLogContext + " exiting from monitor producer events")
}

//func extractEventMetadataFromDeliveryNotification(opaque interface{}) (EventMetadata, bool) {
//	if opaque != nil {
//		if em, ok := opaque.(EventMetadata); ok {
//			return em, ok
//		}
//	}
//
//	return EventMetadata{}, false
//}
