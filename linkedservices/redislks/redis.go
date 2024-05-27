package redislks

import (
	"context"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

type LinkedService struct {
	cfg *Config
	rdb *redis.Client
}

func NewInstanceWithConfig(cfg *Config) (*LinkedService, error) {
	lks := &LinkedService{cfg: cfg}
	return lks, nil
}

func (lks *LinkedService) GetClient() (*redis.Client, error) {

	if lks.rdb == nil {
		lks.rdb = redis.NewClient(&redis.Options{
			Addr:         lks.cfg.Addr,
			Password:     lks.cfg.Passwd,
			DB:           lks.cfg.Db,
			PoolSize:     lks.cfg.PoolSize,
			MaxRetries:   lks.cfg.MaxRetries,
			DialTimeout:  time.Duration(lks.cfg.DialTimeout) * time.Millisecond,
			ReadTimeout:  time.Duration(lks.cfg.ReadTimeout) * time.Millisecond,
			WriteTimeout: time.Duration(lks.cfg.WriteTimeout) * time.Millisecond,
			IdleTimeout:  time.Duration(lks.cfg.IdleTimeout) * time.Millisecond,
		})
	}

	return lks.rdb, nil
}

func (lks *LinkedService) Set(ctx context.Context, key string, value interface{}) error {
	const semLogContext = "redis-lks::set"
	beginOf := time.Now()
	lbls := lks.MetricsLabels(http.MethodPost)
	defer func(start time.Time) {
		_ = lks.setMetrics(start, lbls)
	}(beginOf)

	rdb, err := lks.GetClient()
	if err != nil {
		return err
	}

	var sts *redis.StatusCmd
	switch tv := value.(type) {
	case []byte:
		sts = rdb.Set(ctx, key, tv, lks.cfg.TTL)
	default:
		sts = rdb.Set(ctx, key, value, lks.cfg.TTL)
	}

	err = sts.Err()
	if err == nil {
		lbls[MetricIdStatusCode] = "200"
	}
	return err
}

func (lks *LinkedService) Get(ctx context.Context, key string) (interface{}, error) {

	const semLogContext = "redis-lks::get"
	beginOf := time.Now()
	lbls := lks.MetricsLabels(http.MethodGet)
	defer func(start time.Time) {
		_ = lks.setMetrics(start, lbls)
	}(beginOf)

	rdb, err := lks.GetClient()
	if err != nil {
		return nil, err
	}

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Warn().Str("key", key).Msg(semLogContext + " cached key not found")
			lbls[MetricIdStatusCode] = fmt.Sprint(http.StatusNotFound)
			return nil, nil
		}
		return nil, err
	}

	lbls[MetricIdStatusCode] = fmt.Sprint(http.StatusOK)
	return val, nil
}

const (
	MetricIdStatusCode     = "status-code"
	MetricIdCacheOperation = "operation"
)

func (lks *LinkedService) MetricsLabels(m string) prometheus.Labels {

	metricsLabels := prometheus.Labels{
		MetricIdStatusCode:     fmt.Sprint(http.StatusInternalServerError),
		MetricIdCacheOperation: m,
	}

	return metricsLabels
}

func (lks *LinkedService) setMetrics(begin time.Time, lbls prometheus.Labels) error {
	const semLogContext = "redis-lks::set-metrics"

	var err error
	var g promutil.Group

	cfg := lks.cfg.MetricsCfg
	if cfg.GId != "" && cfg.IsEnabled() {
		g, _, err = cfg.ResolveGroup(nil)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		if cfg.IsCounterEnabled() {
			err = g.SetMetricValueById(cfg.CounterId, 1, lbls)
		}

		if cfg.IsHistogramEnabled() {
			err = g.SetMetricValueById(cfg.HistogramId, time.Since(begin).Seconds(), lbls)
		}
	}

	return err
}
