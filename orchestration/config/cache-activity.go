package config

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-cache-common/cachelks"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

type CacheActivity struct {
	Activity `yaml:",inline" json:",inline"`
}

func (c *CacheActivity) WithName(n string) *CacheActivity {
	c.Nm = n
	return c
}

func (c *CacheActivity) WithActor(n string) *CacheActivity {
	c.Actr = n
	return c
}

func (c *CacheActivity) WithDescription(n string) *CacheActivity {
	c.Cm = n
	return c
}

func (c *CacheActivity) WithExpressionContext(n string) *CacheActivity {
	c.ExprContextName = n
	return c
}

func (c *CacheActivity) WithRefDefinition(n string) *CacheActivity {
	c.Definition = n
	return c
}

func (c *CacheActivity) Dup(newName string) *CacheActivity {

	actNew := CacheActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

func NewCacheActivity() *CacheActivity {
	s := CacheActivity{}
	s.Tp = CacheActivityType
	return &s
}

func NewCacheActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewCacheActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewCacheActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewCacheActivity()
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}

const (
	CacheOperationSet = "set"
	CacheOperationGet = "get"
)

type CacheActivityDefinition struct {
	Operation         string                         `yaml:"op-type,omitempty" json:"op-type,omitempty" mapstructure:"op-type,omitempty"`
	Key               string                         `yaml:"key,omitempty" mapstructure:"key,omitempty" json:"key,omitempty"`
	Namespace         string                         `json:"namespace,omitempty" yaml:"namespace,omitempty" mapstructure:"namespace,omitempty"`
	Ttl               time.Duration                  `yaml:"ttl,omitempty" mapstructure:"ttl,omitempty" json:"ttl,omitempty"`
	LinkedServiceRef  cachelks.CacheLinkedServiceRef `yaml:"broker,omitempty" mapstructure:"broker,omitempty" json:"broker,omitempty"`
	OnResponseActions OnResponseActions              `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func (def *CacheActivityDefinition) IsZero() bool {
	return true
}

func (def *CacheActivityDefinition) WriteToFile(folderName string, fileName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "cache-activity-definition::write-to-file"
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

func UnmarshalCacheActivityDefinition(schemaNamespace, def string, refs DataReferences) (CacheActivityDefinition, error) {
	const semLogContext = "cache-activity-definition::unmarshal"

	var err error
	maDef := CacheActivityDefinition{}

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

	return maDef, nil
}
