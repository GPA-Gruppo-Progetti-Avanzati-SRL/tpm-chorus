package linkedservices

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/linkedservices/redislks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
)

type Config struct {
	RestClient *restclient.Config `json:"rest-client" yaml:"rest-client" mapstructure:"rest-client"`
	Kafka      []kafkalks.Config  `json:"kafka" yaml:"kafka" mapstructure:"kafka"`
	Redis      *redislks.Config   `json:"redis-cache" yaml:"redis-cache" mapstructure:"redis-cache"`
}

func (c *Config) PostProcess() error {

	var err error

	if len(c.Kafka) > 0 {
		for i := range c.Kafka {
			err = c.Kafka[i].PostProcess()
			if err != nil {
				return err
			}
		}
	}

	if err != nil {
		return err
	}

	if c.Redis != nil {
		err = c.Redis.PostProcess()
	}
	if err != nil {
		return err
	}

	return nil
}
