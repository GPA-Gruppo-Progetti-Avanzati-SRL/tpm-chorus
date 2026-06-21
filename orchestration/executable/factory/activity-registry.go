package factory

import (
	"errors"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/executable"
	"github.com/rs/zerolog/log"
)

type ActivityFactory func(item config.Configurable, refs config.DataReferences) (executable.Executable, error)

type ActivityRegistry map[string]ActivityFactory

var activityRegistry ActivityRegistry

func RegisterActivityFactory(tp string, entry ActivityFactory) error {
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

func GetRegisteredActivityFactory(tp string) (ActivityFactory, bool) {
	const semLogContext = "activity-activity_registry::get"

	e, ok := activityRegistry[tp]
	if !ok {
		err := errors.New("activity type not registered")
		log.Warn().Err(err).Str("activity-type", string(tp)).Msg(semLogContext)
		return nil, false
	}

	return e, true
}
