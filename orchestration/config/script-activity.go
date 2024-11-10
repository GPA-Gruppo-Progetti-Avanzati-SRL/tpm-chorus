package config

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
)

type ScriptActivity struct {
	Activity `yaml:",inline" json:",inline"`
	// Engine   string `yaml:"engine,omitempty" json:"engine,omitempty" mapstructure:"engine,omitempty"`
}

func (c *ScriptActivity) WithName(n string) *ScriptActivity {
	c.Nm = n
	return c
}

func (c *ScriptActivity) WithActor(n string) *ScriptActivity {
	c.Actr = n
	return c
}

func (c *ScriptActivity) WithDescription(n string) *ScriptActivity {
	c.Cm = n
	return c
}

func (c *ScriptActivity) WithExpressionContext(n string) *ScriptActivity {
	c.ExprContextName = n
	return c
}

func (c *ScriptActivity) WithRefDefinition(n string) *ScriptActivity {
	c.Definition = n
	return c
}

func (c *ScriptActivity) Dup(newName string) *ScriptActivity {
	actNew := ScriptActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

func NewScriptActivity() *ScriptActivity {
	s := ScriptActivity{}
	s.Tp = ScriptActivityType
	return &s
}

func NewScriptActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewScriptActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewScriptActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewScriptActivity()
	// err := mapstructure.Decode(mp, sa)
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}

type ScriptActivityParam struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty" mapstructure:"name,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
}

type ScriptActivityDefinition struct {
	Engine            string                `yaml:"engine,omitempty" json:"engine,omitempty" mapstructure:"engine,omitempty"`
	Script            string                `yaml:"script,omitempty" json:"script,omitempty" mapstructure:"script,omitempty"`
	ScriptText        []byte                `yaml:"-" json:"-" mapstructure:"-"`
	StdLibModules     []string              `yaml:"std-lib,omitempty" json:"std-lib,omitempty" mapstructure:"std-lib,omitempty"`
	Params            []ScriptActivityParam `yaml:"params,omitempty" json:"params,omitempty" mapstructure:"params,omitempty"`
	OnResponseActions OnResponseActions     `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func (def *ScriptActivityDefinition) IsZero() bool {
	return def.Engine == ""
}

func (def *ScriptActivityDefinition) WriteToFile(folderName string, fileName string) error {
	const semLogContext = "script-activity-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(def)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	outFileName, _ := fileutil.ResolvePath(fn)
	err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func UnmarshalScriptActivityDefinition(def string, refs DataReferences) (ScriptActivityDefinition, error) {
	const semLogContext = "script-activity-definition::unmarshal"

	var err error
	maDef := ScriptActivityDefinition{}

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

	var ok bool
	if maDef.Script != "" {
		maDef.ScriptText, ok = refs.Find(maDef.Script)
	}

	if !ok {
		err = errors.New("cannot find script for activity")
		log.Error().Err(err).Str("script-name", maDef.Script).Msg(semLogContext)
		return maDef, err
	}

	return maDef, nil
}
