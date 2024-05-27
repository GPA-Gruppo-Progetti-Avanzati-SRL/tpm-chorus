package linkedservices

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"tpm-chorus/linkedservices/redislks"
)

type Config struct {
	RestClient *restclient.Config `mapstructure:"rest-client"`
	Kafka      []kafkalks.Config  `mapstructure:"kafka"`
	Redis      *redislks.Config   `mapstructure:"redis-cache"`
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
