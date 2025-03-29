package linkedservices

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelksregistry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/mongolks"
	"github.com/rs/zerolog/log"
)

type ServiceRegistry struct {
	RestClient *restclient.LinkedService
}

var registry ServiceRegistry

func InitRegistry(cfg *Config) error {

	const semLogContext = "service-registry::initialize"
	registry = ServiceRegistry{}
	log.Info().Msg(semLogContext)

	err := initializeRestClientProvider(cfg.RestClient)
	if err != nil {
		return err
	}

	_, err = kafkalks.Initialize(cfg.Kafka)
	if err != nil {
		return err
	}

	if cfg.Redis != nil {
		_, err = cachelksregistry.Initialize(*cfg.Redis)
		if err != nil {
			return err
		}
	}

	_, err = mongolks.Initialize(cfg.MongoDb)
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	const semLogContext = "service-registry::close"
	log.Info().Msg(semLogContext)
	kafkalks.Close()
}

func initializeRestClientProvider(cfg *restclient.Config) error {
	const semLogContext = "service-registry::initialize-http-client-provider"
	log.Info().Msg(semLogContext)
	if cfg != nil {
		lks, err := restclient.NewInstanceWithConfig(cfg)
		if err != nil {
			return err
		}

		registry.RestClient = lks
	}

	return nil
}

func GetRestClientProvider(opts ...restclient.Option) (*restclient.Client, error) {
	const semLogContext = "service-registry::get-http-client-provider"
	if registry.RestClient != nil {
		return registry.RestClient.NewClient(opts...)
	}

	return nil, errors.New(semLogContext + " http client linked service not available")
}
