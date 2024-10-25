package config

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
)

type EchoActivityDefinition struct {
	Message      string `yaml:"message,omitempty" json:"message,omitempty" mapstructure:"message,omitempty"`
	IncludeInHar bool   `yaml:"in-har,omitempty" json:"in-har,omitempty" mapstructure:"in-har,omitempty"`
	WithVars     bool   `yaml:"with-vars,omitempty" json:"with-vars,omitempty" mapstructure:"with-vars,omitempty"`
	WithGoCache  string `yaml:"with-go-cache,omitempty" json:"with-go-cache,omitempty" mapstructure:"with-go-cache,omitempty"`
}

func (d *EchoActivityDefinition) IsZero() bool {
	return d.Message == "" && d.IncludeInHar == false
}

func (d *EchoActivityDefinition) WriteToFile(folderName string, fileName string) error {
	const semLogContext = "echo-activity-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(d)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	outFileName, _ := util.ResolvePath(fn)
	err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func UnmarshalEchoActivityDefinition(def string, refs DataReferences) (EchoActivityDefinition, error) {
	const semLogContext = "echo-activity-definition::unmarshal"

	var err error
	maDef := EchoActivityDefinition{}

	if def != "" {
		data, ok := refs.Find(def)
		if len(data) == 0 || !ok {
			err = errors.New("cannot find activity definition")
			log.Error().Err(err).Msg(semLogContext)
			return maDef, err
		}

		err = yaml.Unmarshal(data, &maDef)
		if err != nil {
			return maDef, err
		}
	}

	return maDef, nil
}
