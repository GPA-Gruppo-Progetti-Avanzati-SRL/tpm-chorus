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
}

func (stat StatsInfo) Clear() StatsInfo {
	stat.NumQueuedMessages = 0
	stat.NumProducedMessages = 0
	stat.NumAcceptedMessages = 0
	stat.NumFailedMessages = 0
	return stat
}

func (stat StatsInfo) IsZero() bool {
	return stat.NumQueuedMessages == 0 && stat.NumProducedMessages == 0 && stat.NumAcceptedMessages == 0 && stat.NumFailedMessages == 0
}

func (stat StatsInfo) IsFail() bool {
	return stat.NumFailedMessages > 0
}

func (stat StatsInfo) IsComplete() bool {
	return stat.NumQueuedMessages == (stat.NumProducedMessages + stat.NumFailedMessages)
}

func (stat StatsInfo) Log(semLogContext string, expectedCompleted bool) {
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

func (stat StatsInfo) IsErr(expectedCompleted bool) error {
	var err error
	if stat.IsFail() {
		err = errors.New("kafka stage with some failures")
	} else if !stat.IsComplete() && expectedCompleted {
		err = errors.New("kafka stage not completed, but should be")
	}

	return err
}
