package linkedservices

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelksregistry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/mongolks"
)

type Config struct {
	RestClient *restclient.Config       `json:"rest-client" yaml:"rest-client" mapstructure:"rest-client"`
	Kafka      []kafkalks.Config        `json:"kafka" yaml:"kafka" mapstructure:"kafka"`
	Redis      *cachelksregistry.Config `json:"cache" yaml:"cache" mapstructure:"cache"`
	MongoDb    []mongolks.Config        `mapstructure:"mongo-db,omitempty"  json:"mongo-db,omitempty" yaml:"mongo-db,omitempty"`
}

func (c *Config) PostProcess() error {

	var err error

	if len(c.MongoDb) > 0 {
		for i := range c.MongoDb {
			err = c.MongoDb[i].PostProcess()
			if err != nil {
				return err
			}
		}
	}

	if len(c.Kafka) > 0 {
		for i := range c.Kafka {
			err = c.Kafka[i].PostProcess()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
