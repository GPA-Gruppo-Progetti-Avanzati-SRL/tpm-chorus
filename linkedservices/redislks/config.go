package redislks

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"time"
)

type Config struct {
	Addr         string                          `mapstructure:"addr"`
	Passwd       string                          `mapstructure:"key"`
	Db           int                             `mapstructure:"db"`
	TTL          time.Duration                   `mapstructure:"ttl"`
	PoolSize     int                             `mapstructure:"poolSize"`
	MaxRetries   int                             `mapstructure:"maxRetries"`
	DialTimeout  int                             `mapstructure:"dialTimeout"`
	ReadTimeout  int                             `mapstructure:"readTimeout"`
	WriteTimeout int                             `mapstructure:"writeTimeout"`
	IdleTimeout  int                             `mapstructure:"idleTimeout"`
	MetricsCfg   promutil.MetricsConfigReference `yaml:"metrics,omitempty" mapstructure:"metrics,omitempty" json:"metrics,omitempty"`
}

func (c *Config) PostProcess() error {
	return nil
}
