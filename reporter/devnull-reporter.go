package reporter

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/rs/zerolog/log"
	"sync"
)

type DevNullReporter struct {
	cfg         *Config
	qmu         sync.Mutex
	workQueue   chan *wfcase.WfCase
	queuedItems int

	wg *sync.WaitGroup
}

func (drep *DevNullReporter) Start() error {
	drep.wg.Add(1)
	go drep.doWorkLoop()

	return nil
}

func (drep *DevNullReporter) Stop() {
	close(drep.workQueue)
}

func (drep *DevNullReporter) doWorkLoop() {
	const semLogContext = "dummy-reporter::work-loop"
	log.Info().Msg(semLogContext + " starting...")

	for _ = range drep.workQueue {
		drep.done()
	}

	drep.wg.Done()
}

func (drep *DevNullReporter) done() {
	drep.qmu.Lock()
	defer drep.qmu.Unlock()

	drep.queuedItems--
}

func (drep *DevNullReporter) Report(req *wfcase.WfCase) error {
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
