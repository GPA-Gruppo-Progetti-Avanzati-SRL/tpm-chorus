package orchestration

import (
	"errors"

	"github.com/rs/zerolog/log"
)

type ActivityFactoryRegistryEntry struct {
}

type ActivityRegistry map[string]ActivityFactoryRegistryEntry

var activityRegistry ActivityRegistry

func RegisterActivityFactory(tp string, entry ActivityFactoryRegistryEntry) error {
	const semLogContext = "activity-activity_registry::add"

	if activityRegistry == nil {
		activityRegistry = make(ActivityRegistry)
	}

	if _, ok := activityRegistry[tp]; ok {
		log.Warn().Str("activity-type", string(tp)).Msg(semLogContext)
	} else {
		activityRegistry[tp] = entry
		return nil
	}

	return nil
}

func GetRegisteredActivityFactory(tp string) (ActivityFactoryRegistryEntry, bool) {
	const semLogContext = "activity-activity_registry::get"

	e, ok := activityRegistry[tp]
	if !ok {
		err := errors.New("activity type not registered")
		log.Warn().Err(err).Str("activity-type", string(tp)).Msg(semLogContext)
		return ActivityFactoryRegistryEntry{}, false
	}

	return e, true
}
