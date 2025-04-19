package config

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/jsonschemaregistry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type JsonSchemaActivity struct {
	Activity `yaml:",inline" json:",inline"`
}

func (c *JsonSchemaActivity) WithName(n string) *JsonSchemaActivity {
	c.Nm = n
	return c
}

func (c *JsonSchemaActivity) WithActor(n string) *JsonSchemaActivity {
	c.Actr = n
	return c
}

func (c *JsonSchemaActivity) WithDescription(n string) *JsonSchemaActivity {
	c.Cm = n
	return c
}

func (c *JsonSchemaActivity) WithExpressionContext(n string) *JsonSchemaActivity {
	c.ExprContextName = n
	return c
}

func (c *JsonSchemaActivity) WithRefDefinition(n string) *JsonSchemaActivity {
	c.Definition = n
	return c
}

func (c *JsonSchemaActivity) Dup(newName string) *JsonSchemaActivity {

	actNew := JsonSchemaActivity{
		Activity: c.Activity.Dup(newName),
	}

	return &actNew
}

func NewJsonSchemaActivity() *JsonSchemaActivity {
	s := JsonSchemaActivity{}
	s.Tp = JsonSchemaActivityType
	return &s
}

func NewJsonSchemaActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewJsonSchemaActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func NewJsonSchemaActivityFromYAML(b []byte /* mp interface{}*/) (Configurable, error) {
	sa := NewJsonSchemaActivity()
	err := yaml.Unmarshal(b, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}

type JsonSchemaRef struct {
	SchemaFile  string `yaml:"schema-file,omitempty" json:"schema-file,omitempty" mapstructure:"schema-file,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
	Guard       string `yaml:"guard,omitempty" json:"guard,omitempty" mapstructure:"guard,omitempty"`
}

type JsonSchemaActivityDefinition struct {
	SchemaRef         []JsonSchemaRef   `yaml:"schemas,omitempty" json:"schemas,omitempty" mapstructure:"schemas,omitempty"`
	OnResponseActions OnResponseActions `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
}

func (def *JsonSchemaActivityDefinition) IsZero() bool {
	return len(def.SchemaRef) == 0
}

func (def *JsonSchemaActivityDefinition) WriteToFile(folderName string, fileName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "json-schema-definition::write-to-file"
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

func UnmarshalJsonSchemaActivityDefinition(schemaNamespace, def string, refs DataReferences) (JsonSchemaActivityDefinition, error) {
	const semLogContext = "json-schema-activity-definition::unmarshal"

	var err error
	maDef := JsonSchemaActivityDefinition{}

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

	for _, ref := range maDef.SchemaRef {
		data, ok := refs.Find(ref.SchemaFile)
		if len(data) == 0 || !ok {
			err = errors.New("cannot find schema file")
			log.Error().Err(err).Str("def", ref.SchemaFile).Msg(semLogContext)
			return maDef, err
		}

		err = jsonschemaregistry.Register(schemaNamespace, ref.SchemaFile, data)
		if err != nil {
			log.Error().Err(err).Str("def", ref.SchemaFile).Msg(semLogContext)
			return maDef, err
		}
	}

	return maDef, nil
}
