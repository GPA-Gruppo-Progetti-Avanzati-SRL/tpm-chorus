package config

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type TransformActivityDefinition struct {
	Transforms        []transform.TransformReference `yaml:"transforms,omitempty"  json:"transforms,omitempty" mapstructure:"transforms,omitempty"`
	OnResponseActions []OnResponseAction             `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func UnmarshalTransformActivityDefinition(def string, refs DataReferences) (TransformActivityDefinition, error) {
	const semLogContext = "transform-activity-definition::unmarshal"

	var err error
	maDef := TransformActivityDefinition{}
	data, ok := refs.Find(def)
	if len(data) == 0 || !ok {
		err = errors.New("cannot find transform activity definition")
		log.Error().Err(err).Msg(semLogContext)
		return maDef, err
	}

	err = yaml.Unmarshal(data, &maDef)
	if err != nil {
		return maDef, err
	}

	for i, xForm := range maDef.Transforms {
		var b []byte
		switch xForm.Typ {
		case XFormKazaam:
			err = registerKazaamXForm(refs, xForm)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return maDef, err
			}
		case XFormTemplate:
			b, err = loadTemplateXForm(refs, xForm.DefinitionRef)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return maDef, err
			}
			maDef.Transforms[i].Data = b
		case XFormMerge:
			b, err = loadMergeXForm(refs, xForm.DefinitionRef)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return maDef, err
			}
			maDef.Transforms[i].Data = b
		default:
			err = fmt.Errorf("unknown xform type: %s", xForm.Typ)
			log.Error().Err(err).Msg(semLogContext)
			return maDef, err
		}

	}
	return maDef, nil
}

func loadMergeXForm(refs DataReferences, mergeRef string) ([]byte, error) {
	const semLogContext = "transform-activity-definition::load-merge-xform"
	trasDef, _ := refs.Find(mergeRef)
	if len(trasDef) == 0 {
		err := fmt.Errorf("cannot find merge %s definition", mergeRef)
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return trasDef, nil
}

func loadTemplateXForm(refs DataReferences, templateRef string) ([]byte, error) {
	const semLogContext = "transform-activity-definition::load-template-xform"
	trasDef, _ := refs.Find(templateRef)
	if len(trasDef) == 0 {
		err := fmt.Errorf("cannot find template %s definition", templateRef)
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return trasDef, nil
}

func registerKazaamXForm(refs DataReferences, xform transform.TransformReference) error {
	const semLogContext = "transform-activity-definition::register-kazaam-xform"

	if xform.Typ != XFormKazaam {
		return nil
	}

	tReg := transform.GetRegistry()
	if tReg == nil {
		err := errors.New("transformation registry not initialized")
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	trasDef, _ := refs.Find(xform.DefinitionRef)
	if len(trasDef) == 0 {
		err := fmt.Errorf("cannot find transformation %s definition from %s", xform.Id, xform.DefinitionRef)
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	xform.Data = trasDef
	err := tReg.AddTransformation(xform)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	/*
		trsf := transform.Config{}
		err := yaml.Unmarshal(trasDef, &trsf)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		err = tReg.Add(trsf)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}
	*/

	return nil
}
