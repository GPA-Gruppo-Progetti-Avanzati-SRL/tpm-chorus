package reporter

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/jsonmask"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"sync"
)

const CountHarMessagesMetricId = "har-messages"

type KafkaConfig struct {
	Topic string `mapstructure:"topic"`
}

type KafkaOption func(o *KafkaConfig)

func WithTopic(t string) KafkaOption {
	return func(o *KafkaConfig) {
		o.Topic = t
	}
}

type KafkaReporter struct {
	cfg *Config
	qmu sync.Mutex

	metricsRegistry promutil.Group
	jsonMask        *jsonmask.JsonMask
	workQueue       chan *wfcase.WfCase
	queuedItems     int
	kProducer       *kafka.Producer
	wg              *sync.WaitGroup
}

func (dw *KafkaReporter) Start() error {

	kp, err := kafkalks.NewKafkaProducer(context.Background(), "default", "") // .NewKafkaProducer(context.Background(), dw.cfg.Kafka.Topic)
	if err != nil {
		return err
	}

	dw.kProducer = kp
	dw.wg.Add(1)

	go dw.monitorProducerEvents()
	go dw.doWorkLoop()

	return nil
}

func (w *KafkaReporter) Stop() {
	close(w.workQueue)
}

func (kr *KafkaReporter) doWorkLoop() {
	const semLogContext = "kafka-reporter::work-loop"
	log.Info().Msg(semLogContext + " starting...")

	for wfc := range kr.workQueue {

		metricLabels := kr.MetricsLabels()

		wfc.ShowBreadcrumb()
		wfc.ShowVars(false)

		var b []byte
		var err error
		switch kr.cfg.DetailLevel {
		case wfcase.ReportLogHAR:
			fallthrough
		case wfcase.ReportLogHARRequest:
			har := wfc.GetHarData(kr.cfg.DetailLevel, kr.jsonMask)
			b, err = json.Marshal(har)
		default:
			err = errors.New("invalid detail level")
			log.Error().Err(err).Str("lev", string(kr.cfg.DetailLevel)).Msg(semLogContext)
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			wfc.ShowBreadcrumb()
			metricLabels[MetricIdStatusCode] = "500"
			_ = kr.SetMetrics(metricLabels)
		} else {
			//Recupero requestId dal wfc per utilizzarlo come Key del message kafka
			requestId := wfc.GetHeaderFromContext(config.InitialRequestResolverExpressionScope, "requestId")
			/*			reqEntry, ok := wfc.Entries["request"]
						if ok {
							requestId = reqEntry.Request.Headers.GetFirst("requestId").Value
						}*/
			err = kr.produce(requestId, b)
			if err != nil {
				metricLabels[MetricIdStatusCode] = "500"
				_ = kr.SetMetrics(metricLabels)
			}
		}

		kr.done()
	}

	// Added the close on the producer...
	if kr.kProducer != nil {
		kr.kProducer.Close()
	}

	kr.wg.Done()
}

func (kr *KafkaReporter) produce(keyValue string, b []byte) error {

	const semLogContext = "kafka-reporter::produce"

	var err error
	err = kr.kProducer.Produce(&kafka.Message{
		Key:            []byte(keyValue),
		TopicPartition: kafka.TopicPartition{Topic: &kr.cfg.Kafka.Topic, Partition: kafka.PartitionAny},
		Value:          b,
	}, nil)

	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	return err
}

func (kr *KafkaReporter) done() {
	kr.qmu.Lock()
	defer kr.qmu.Unlock()

	kr.queuedItems--
}

func (kr *KafkaReporter) monitorProducerEvents() {

	const semLogContext = "kafka-reporter::monitor-producer-events"
	log.Info().Msg(semLogContext + " starting...")

	exitFromLoop := false
	for e := range kr.kProducer.Events() {

		metricLabels := kr.MetricsLabels()

		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Error().Err(ev.TopicPartition.Error).Msg("Delivery failed")
				metricLabels[MetricIdStatusCode] = "500"
				kr.SetMetrics(metricLabels)
			} else {
				metricLabels[MetricIdStatusCode] = "200"
				kr.SetMetrics(metricLabels)
			}
		case kafka.Error:
			log.Error().Bool("is-retriable", ev.IsRetriable()).Bool("is-fatal", ev.IsFatal()).Interface("error", ev.Error()).Interface("code", ev.Code()).Interface("text", ev.Code().String()).Msg(semLogContext)
			metricLabels[MetricIdStatusCode] = "503"
			kr.SetMetrics(metricLabels)
		}

		if exitFromLoop {
			break
		}
	}

	log.Info().Msg("stop monitoring producer events")
}

func (kr *KafkaReporter) Report(req *wfcase.WfCase) error {
	kr.qmu.Lock()
	defer kr.qmu.Unlock()

	if kr.queuedItems >= (kr.cfg.QueueSize - (kr.cfg.QueueSize*20)/100) {
		return errors.New("too many requests")
	}

	kr.queuedItems++
	kr.workQueue <- req
	log.Trace().Int("queue-length", kr.queuedItems).Msg("accepting kafka reporting case")
	return nil
}

/*
	func (drep *KafkaReporter) MetricsGroup() (promutil.Group, bool, error) {
		mCfg := drep.cfg.RefMetricsConfig

		var g promutil.Group
		var err error
		var ok bool
		if mCfg.IsEnabled() {
			if mCfg.GId != DefaultMetricsEmbeddedGroupId {
				g, err = promutil.GetGroup(mCfg.GId)
			} else {
				g = drep.metricsRegistry
			}
			if err == nil {
				ok = true
			}
		}

		return g, ok, err
	}
*/

func (drep *KafkaReporter) SetMetrics(lbls prometheus.Labels) error {
	const semLogContext = "kafka-reporter::set-metrics"
	cfg := drep.cfg.RefMetricsConfig
	if cfg.IsEnabled() {
		g, _, err := cfg.ResolveGroup(drep.metricsRegistry)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		if cfg.IsCounterEnabled() {
			g.SetMetricValueById(cfg.CounterId, 1, lbls)
		}
	}

	return nil
}

func (a *KafkaReporter) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdStatusCode: "500",
	}

	return metricsLabels
}
