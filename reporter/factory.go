package reporter

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/jsonmask"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
)

type Reporter interface {
	Start() error
	Stop()
	Report(req *wfcase.WfCase) error
}

const DefaultReporterName = "default"

var registry = make(map[string]Reporter)

func GetReporter(n string) (Reporter, error) {
	const semLogContext = "reporter::get-reporter"
	rep, ok := registry[strings.ToLower(n)]
	if !ok {
		err := errors.New("reporter not found by name")
		log.Error().Err(err).Str("name", n).Msg(semLogContext)
		return nil, err
	}

	return rep, nil
}

func NewReporter(n string, cfg *Config, wg *sync.WaitGroup) (Reporter, error) {

	const semLogContext = "reporter::new-reporter"
	var rep Reporter

	if cfg.ReporterType == "devnull" {
		w := &DevNullReporter{}
		rep = w
		registry[strings.ToLower(n)] = rep
		return rep, nil
	}

	log.Info().Int("reporter-queue-size", cfg.QueueSize).Msg(semLogContext)

	wq := make(chan *wfcase.WfCase, cfg.QueueSize)

	var jm *jsonmask.JsonMask
	var err error
	if cfg.PiiFile != "" {
		jm, err = jsonmask.NewJsonMask(jsonmask.FromFileName(cfg.PiiFile))
		if err != nil {
			log.Error().Err(err).Str("fn", cfg.PiiFile).Msg(semLogContext + " error reading pii config data")
		}
	}

	var mr promutil.Group
	if cfg.RefMetricsConfig == nil {
		if cfg.MetricsConfig.Namespace != "" && cfg.MetricsConfig.Subsystem != "" && len(cfg.MetricsConfig.Collectors) > 0 {
			mr, err = promutil.InitGroup(cfg.MetricsConfig)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext + " initialization metrics data")
			}

			cfg.RefMetricsConfig = &promutil.MetricsConfigReference{
				GId:       DefaultMetricsEmbeddedGroupId,
				CounterId: DefaultCounterId,
			}
		} else {
			log.Info().Msg(semLogContext + " metrics not configured for har reporting")
			cfg.RefMetricsConfig = &promutil.MetricsConfigReference{
				GId:       "-",
				CounterId: DefaultCounterId,
			}
		}
	} else {
		log.Info().Msg(semLogContext + " using referenced metrics configuration")
	}

	switch cfg.ReporterType {
	case "dummy":
		w := &DummyReporter{cfg: cfg, wg: wg, workQueue: wq, metricsRegistry: mr}
		w.jsonMask = jm
		rep = w

	//case "devnull":
	//	w := &DevNullReporter{ /*cfg: cfg, wg: wg, workQueue: wq*/ }
	//	rep = w

	case "kafka":
		w := &KafkaReporter{cfg: cfg, wg: wg, workQueue: wq, metricsRegistry: mr}
		w.jsonMask = jm
		rep = w

	default:
		err = errors.New("unsupported reporter type")
		log.Error().Err(err).Str("type", cfg.ReporterType).Msg(semLogContext)
	}

	registry[strings.ToLower(n)] = rep
	return rep, err
}
