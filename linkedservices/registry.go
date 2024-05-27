package linkedservices

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"github.com/rs/zerolog/log"
	"tpm-chorus/linkedservices/redislks"
)

type ServiceRegistry struct {
	RestClient *restclient.LinkedService
	kafka      *kafkalks.LinkedService
	redis      *redislks.LinkedService
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

	err = initializeRedisCache(cfg.Redis)
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

/*
 * Redis cache Initialization
 */

func initializeRedisCache(cfg *redislks.Config) error {
	const semLogContext = "service-registry::initialize-redis-cache-provider"
	log.Info().Msg(semLogContext)
	if cfg != nil {
		lks, err := redislks.NewInstanceWithConfig(cfg)
		if err != nil {
			return err
		}

		registry.redis = lks
	}

	return nil
}

func GetRedisCacheLinkedService() (*redislks.LinkedService, error) {
	return registry.redis, nil
}
