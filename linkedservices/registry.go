package linkedservices

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/redislks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-client/restclient"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/kafkalks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/mongolks"
	"github.com/rs/zerolog/log"
)

type ServiceRegistry struct {
	RestClient *restclient.LinkedService
	redis      []*redislks.LinkedService
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

/*
 * Redis cache Initialization
 */

func initializeRedisCache(cfg []redislks.Config) error {
	const semLogContext = "service-registry::initialize-redis-cache-provider"
	log.Info().Msg(semLogContext)
	for _, c := range cfg {
		lks, err := redislks.NewInstanceWithConfig(c)
		if err != nil {
			return err
		}

		registry.redis = append(registry.redis, lks)
	}

	return nil
}

func GetRedisCacheLinkedService(name string) (*redislks.LinkedService, error) {

	if (name == redislks.RedisDefaultBrokerName || name == "") && len(registry.redis) == 1 {
		return registry.redis[0], nil
	}

	for _, r := range registry.redis {
		if r.Name() == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("cannot find redis cache by name [%s]", name)
}
