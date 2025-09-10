package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const (
	LoopControlFLowFor = "for"
)

type LoopControlFlowDefinition struct {
	Typ            string                    `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"`
	Start          string                    `yaml:"start,omitempty" json:"start,omitempty" mapstructure:"start,omitempty"`
	End            string                    `yaml:"end,omitempty" json:"end,omitempty" mapstructure:"end,omitempty"`
	Step           string                    `yaml:"step,omitempty" json:"step,omitempty" mapstructure:"step,omitempty"`
	BreakCondition string                    `yaml:"break-on,omitempty" json:"break-on,omitempty" mapstructure:"break-on,omitempty"`
	XForm          xforms.TransformReference `yaml:"x-form,omitempty"  json:"x-form,omitempty" mapstructure:"x-form,omitempty"`
}

type LoopActivityDefinition struct {
	OrchestrationId   string                    `yaml:"orchestration-id,omitempty" json:"orchestration-id,omitempty" mapstructure:"orchestration-id,omitempty"`
	ControlFlow       LoopControlFlowDefinition `yaml:"control-flow,omitempty" json:"control-flow,omitempty" mapstructure:"control-flow,omitempty"`
	OnResponseActions OnResponseActions         `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func (def *LoopActivityDefinition) WriteToFile(folderName string, fileName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "loop-activity-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(def)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	err = fileutil.WriteFile(fn, b, os.ModePerm, writeOpts...)
	//outFileName, _ := fileutil.ResolvePath(fn)
	//err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func UnmarshalLoopActivityDefinition(def string, refs DataReferences) (LoopActivityDefinition, error) {
	const semLogContext = "loop-activity-definition::unmarshal"

	var err error
	maDef := LoopActivityDefinition{}

	if def != "" {
		data, ok := refs.Find(def)
		if len(data) == 0 || !ok {
			err = errors.New("cannot find activity definition")
			log.Error().Err(err).Str("def", def).Msg(semLogContext)
			return maDef, err
		}

		err = yaml.Unmarshal(data, &maDef)
		if err != nil {
			return maDef, err
		}
	}

	if maDef.ControlFlow.Typ == "" {
		maDef.ControlFlow.Typ = LoopControlFLowFor
	}

	switch maDef.ControlFlow.XForm.Typ {
	case XFormKazaamDynamic:
		b, err := loadKazaamXForm(refs, maDef.ControlFlow.XForm)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return maDef, err
		}
		maDef.ControlFlow.XForm.Data = b

	case XFormKazaam:
		err = registerKazaamXForm(refs, maDef.ControlFlow.XForm)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return maDef, err
		}

	case XFormJQ:
		err = registerJQXForm(refs, maDef.ControlFlow.XForm)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return maDef, err
		}

	default:
		err = fmt.Errorf("unsupported xform type: %s", maDef.ControlFlow.XForm.Typ)
		log.Error().Err(err).Msg(semLogContext)
		return maDef, err
	}

	return maDef, nil
}
