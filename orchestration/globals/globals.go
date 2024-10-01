package globals

import (
	"context"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks/gocachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelksregistry"
	"github.com/rs/zerolog/log"
	"time"
)

const (
	GlobalVarsLinkedServiceName = "global-vars"
)

var LinkedServiceRef = cachelks.CacheLinkedServiceRef{
	Name: GlobalVarsLinkedServiceName,
	Typ:  gocachelks.GoCacheLinkedServiceType,
}

func GlobalVars() (map[string]interface{}, error) {
	const semLogContext = "globals::global-vars"

	if LinkedServiceRef.Typ != gocachelks.GoCacheLinkedServiceType {
		err := errors.New("GlobalVars returned only if in the go-cache linked service")
		log.Warn().Err(err).Msg(semLogContext)
		return nil, err
	}

	lks, err := cachelksregistry.GetLinkedServiceOfType(LinkedServiceRef.Typ, LinkedServiceRef.Name)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	goCacheLks, ok := lks.(*gocachelks.LinkedService)
	if !ok {
		err := errors.New("cannot cast linked service to go-cache linked service")
		log.Warn().Err(err).Msg(semLogContext)
		return nil, err
	}

	items := make(map[string]interface{}, goCacheLks.Size())
	for n, v := range goCacheLks.Items() {
		items[n] = v.Object
	}

	return items, nil
}

func GetGlobalVar(ns, name, defaultValue string) (interface{}, error) {
	const semLogContext = "globals::get-global-var"

	lks, err := cachelksregistry.GetLinkedServiceOfType(LinkedServiceRef.Typ, LinkedServiceRef.Name)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	opts := cachelks.CacheOptions{
		Namespace: ns,
	}
	v, err := lks.Get(context.Background(), name, opts)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return v, nil
}

func SetGlobalVar(ns, name string, v interface{}, ttl time.Duration) error {
	const semLogContext = "globals::set-global-var"

	lks, err := cachelksregistry.GetLinkedServiceOfType(LinkedServiceRef.Typ, LinkedServiceRef.Name)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	opts := cachelks.CacheOptions{
		Namespace: ns,
		Ttl:       ttl,
	}
	err = lks.Set(context.Background(), name, v, opts)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}
