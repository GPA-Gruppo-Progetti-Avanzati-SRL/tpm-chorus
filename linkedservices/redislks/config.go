package redislks

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"time"
)

const (
	RedisDefaultBrokerName               = "default"
	RedisDefaultDbIndex                  = 0
	RedisUseLinkedServiceConfiguredIndex = -1
)

type Config struct {
	Name         string                          `mapstructure:"name,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	Addr         string                          `mapstructure:"addr,omitempty" json:"addr,omitempty" yaml:"addr,omitempty"`
	Passwd       string                          `mapstructure:"key,omitempty" json:"passwd,omitempty" yaml:"key,omitempty"`
	Db           int                             `mapstructure:"db,omitempty" json:"db,omitempty" yaml:"db,omitempty"`
	TTL          time.Duration                   `mapstructure:"ttl,omitempty" json:"ttl,omitempty" yaml:"ttl,omitempty"`
	PoolSize     int                             `mapstructure:"poolSize,omitempty" json:"poolSize,omitempty" yaml:"poolSize,omitempty"`
	MaxRetries   int                             `mapstructure:"maxRetries,omitempty" json:"maxRetries,omitempty" yaml:"maxRetries,omitempty"`
	DialTimeout  int                             `mapstructure:"dialTimeout,omitempty" json:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty"`
	ReadTimeout  int                             `mapstructure:"readTimeout,omitempty" json:"readTimeout,omitempty" yaml:"readTimeout,omitempty"`
	WriteTimeout int                             `mapstructure:"writeTimeout,omitempty" json:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty"`
	IdleTimeout  int                             `mapstructure:"idleTimeout,omitempty" json:"idleTimeout,omitempty" yaml:"idleTimeout,omitempty"`
	MetricsCfg   promutil.MetricsConfigReference `yaml:"metrics,omitempty" mapstructure:"metrics,omitempty" json:"metrics,omitempty"`
}

func (c *Config) PostProcess() error {
	return nil
}
