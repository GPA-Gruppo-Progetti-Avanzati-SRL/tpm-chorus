package reporter

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/jsonmask"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"io/fs"
	"os"
	"sync"
)

const (
	SemLogDocClass = "doc-class"
	SemLogPkgId    = "pkg-id"
	SemLogNumDocs  = "num-docs"
)

type DummyReporterConfig struct {
	Filename string `mapstructure:"file-name"`
}

type DummyReporter struct {
	cfg *Config

	metricsRegistry promutil.Group
	jsonMask        *jsonmask.JsonMask
	qmu             sync.Mutex
	workQueue       chan *wfcase.WfCase
	queuedItems     int

	wg *sync.WaitGroup
}

func (drep *DummyReporter) Start() error {
	drep.wg.Add(1)
	go drep.doWorkLoop()

	return nil
}

func (drep *DummyReporter) Stop() {
	close(drep.workQueue)
}

func (drep *DummyReporter) doWorkLoop() {
	const semLogContext = "dummy-reporter::work-loop"
	log.Info().Msg(semLogContext + " starting...")

	for wfc := range drep.workQueue {

		metricLabels := drep.MetricsLabels()

		wfc.ShowBreadcrumb()
		wfc.Vars.ShowVars(true)

		var b []byte
		var err error
		switch drep.cfg.DetailLevel {
		case wfcase.ReportLogHAR:
			fallthrough
		case wfcase.ReportLogHARRequest:
			har := wfc.GetHarData(drep.cfg.DetailLevel, drep.jsonMask)
			b, err = jsoniter.Marshal(har)
		default:
			err = errors.New("invalid detail level")
			log.Error().Err(err).Str("lev", string(drep.cfg.DetailLevel)).Msg(semLogContext)
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			wfc.ShowBreadcrumb()
			metricLabels[MetricIdStatusCode] = "500"
			drep.SetMetrics(metricLabels)
		} else {
			drep.write2File(b)
			metricLabels[MetricIdStatusCode] = "200"
			drep.SetMetrics(metricLabels)
		}

		drep.done()
	}

	drep.wg.Done()
}

func (drep *DummyReporter) write2File(b []byte) {
	const semLogContext = "dummy-reporter::write-2-file"
	if drep.cfg.Dummy != nil && drep.cfg.Dummy.Filename != "" {
		log.Trace().Str("filename", drep.cfg.Dummy.Filename).Interface("mode", drep.cfg.DetailLevel).Msg(semLogContext + " writing log")
		err := os.WriteFile(drep.cfg.Dummy.Filename, b, fs.ModePerm)
		if err != nil {
			log.Error().Err(err).Str("filename", drep.cfg.Dummy.Filename).Msg(semLogContext + " error writing to har log")
		}
	}
}

func (drep *DummyReporter) done() {
	drep.qmu.Lock()
	defer drep.qmu.Unlock()

	drep.queuedItems--
}

func (drep *DummyReporter) Report(req *wfcase.WfCase) error {
	drep.qmu.Lock()
	defer drep.qmu.Unlock()

	if drep.queuedItems >= (drep.cfg.QueueSize - (drep.cfg.QueueSize*20)/100) {
		return errors.New("too many requests")
	}

	drep.queuedItems++
	drep.workQueue <- req
	log.Trace().Int("queue-length", drep.queuedItems).Msg("accepting reporting case")
	return nil
}

/*
func (drep *DummyReporter) MetricsGroup() (promutil.Group, bool, error) {
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

func (drep *DummyReporter) SetMetrics(lbls prometheus.Labels) error {
	const semLogContext = "dummy-reporter::set-metrics"
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

const (
	MetricIdStatusCode = "status-code"
)

func (a *DummyReporter) MetricsLabels() prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdStatusCode: "500",
	}

	return metricsLabels
}
