package jsonschemaregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"github.com/santhosh-tekuri/jsonschema"
	"regexp"
	"strings"
)

const (
	JsonSchemaUnknownCause    = "unknown"
	JsonSchemaMissingProperty = "missing-property"
	JsonSchemaInvalidValue    = "invalid-value"

	MissingPropertyMessageCause         = "missing properties: "
	MissingPropertyMessageRegexpPattern = "missing properties: \"([A-Za-z\\-]+)[\\s\\n\\r]*\""

	SchemaPtrRequired            = "#/required"
	SchemaPtrProperties          = "#/properties/"
	SchemaPtrPropertiesEnum      = "/enum"
	SchemaPtrPropertiesMaxLength = "/maxLength"
	SchemaPtrPropertiesMinLength = "/minLength"
	SchemaPtrPropertiesPattern   = "/pattern"
)

var MissingPropertyCauseRegexp = regexp.MustCompile(MissingPropertyMessageRegexpPattern)

type SchemaErrorCause struct {
	Typ  string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"`
	Name string `yaml:"name,omitempty" json:"name,omitempty" mapstructure:"name,omitempty"`
	Msg  string `yaml:"msg,omitempty" json:"msg,omitempty" mapstructure:"msg,omitempty"`
}

type SchemaError struct {
	Message string             `yaml:"message,omitempty" json:"message,omitempty" mapstructure:"message,omitempty"`
	Causes  []SchemaErrorCause `yaml:"causes,omitempty" json:"causes,omitempty" mapstructure:"causes,omitempty"`
}

func (schErr SchemaError) Error() string {
	return schErr.ToJson()
}

func (schErr SchemaError) ToJson() string {
	const semLogContext = "json-schema-registry::error marshalling schema error"
	b, err := json.Marshal(schErr)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling schema error")
		return ""
	}

	return string(b)
}

func NewSchemaErrorFromError(schema *jsonschema.Schema, err error) (SchemaError, bool) {
	const semLogContext = "json-schema-registry::new-schema-error-from-error"
	var validationErr *jsonschema.ValidationError
	var schemaErr SchemaError
	if errors.As(err, &validationErr) {
		schemaErr.Message = validationErr.Message
		for _, cause := range validationErr.Causes {
			var c SchemaErrorCause
			switch m := cause.SchemaPtr; {
			case strings.HasPrefix(m, SchemaPtrRequired):
				if strings.HasPrefix(cause.Message, MissingPropertyMessageCause) {
					c = SchemaErrorCause{Typ: JsonSchemaMissingProperty, Name: util.ExtractCapturedGroupIfMatch(MissingPropertyCauseRegexp, cause.Message)}
				} else {
					log.Warn().Str("cause-msg", cause.Message).Str("schema-ptr", m).Msg(semLogContext)
				}
			case strings.HasPrefix(m, SchemaPtrProperties) && strings.HasSuffix(m, SchemaPtrPropertiesEnum):
				c = SchemaErrorCause{Typ: JsonSchemaInvalidValue, Name: strings.TrimSuffix(strings.TrimPrefix(m, SchemaPtrProperties), SchemaPtrPropertiesEnum), Msg: cause.Message}
			case strings.HasPrefix(m, SchemaPtrProperties) && strings.HasSuffix(m, SchemaPtrPropertiesMaxLength):
				c = SchemaErrorCause{Typ: JsonSchemaInvalidValue, Name: strings.TrimSuffix(strings.TrimPrefix(m, SchemaPtrProperties), SchemaPtrPropertiesMaxLength), Msg: cause.Message}
			case strings.HasPrefix(m, SchemaPtrProperties) && strings.HasSuffix(m, SchemaPtrPropertiesMinLength):
				c = SchemaErrorCause{Typ: JsonSchemaInvalidValue, Name: strings.TrimSuffix(strings.TrimPrefix(m, SchemaPtrProperties), SchemaPtrPropertiesMinLength), Msg: cause.Message}
			case strings.HasPrefix(m, SchemaPtrProperties) && strings.HasSuffix(m, SchemaPtrPropertiesPattern):
				c = SchemaErrorCause{Typ: JsonSchemaInvalidValue, Name: strings.TrimSuffix(strings.TrimPrefix(m, SchemaPtrProperties), SchemaPtrPropertiesPattern), Msg: cause.Message}
			default:
				c = SchemaErrorCause{Typ: JsonSchemaUnknownCause, Msg: m}
			}
			schemaErr.Causes = append(schemaErr.Causes, c)
		}
		return schemaErr, true
	}

	return schemaErr, false
}

type SchemaRegistry struct {
	schemaMap map[string]*jsonschema.Schema
	compiler  *jsonschema.Compiler
}

var theRegistry SchemaRegistry

func (r *SchemaRegistry) IsZero() bool {
	return r.schemaMap == nil
}

func initializeRegistry() error {
	if theRegistry.schemaMap == nil {
		theRegistry.schemaMap = make(map[string]*jsonschema.Schema)
		theRegistry.compiler = jsonschema.NewCompiler()
	}

	return nil
}

func Register(namespace string, refSchema string, data []byte) error {
	const semLogContext = "json-schema-registry::read-schema"
	var err error
	if refSchema == "" {
		return nil
	}

	if theRegistry.IsZero() {
		_ = initializeRegistry()
	}

	entryName := strings.ToLower(refSchema)
	if namespace != "" {
		entryName = strings.Join([]string{strings.ToLower(namespace), strings.ToLower(refSchema)}, "#")
	} else {
		err = errors.New("namespace is missing")
		log.Warn().Err(err).Msg(semLogContext)
	}

	var schema *jsonschema.Schema
	var ok bool
	if schema, ok = theRegistry.schemaMap[entryName]; !ok {
		if err = theRegistry.compiler.AddResource(refSchema, strings.NewReader(string(data))); err != nil {
			log.Error().Err(err).Str("schema", refSchema).Msg(semLogContext)
			return err
		}

		schema, err = theRegistry.compiler.Compile(refSchema)
		if err != nil {
			log.Error().Err(err).Str("schema", refSchema).Msg(semLogContext)
			return err
		}

		theRegistry.schemaMap[entryName] = schema
	}

	return nil
}

func Validate(namespace, refSchema string, data []byte) error {
	const semLogContext = "json-schema-registry::validate-schema"

	entryName := strings.Join([]string{strings.ToLower(namespace), strings.ToLower(refSchema)}, "#")

	var ok bool
	var err error
	var schema *jsonschema.Schema
	if schema, ok = theRegistry.schemaMap[entryName]; !ok {
		err = fmt.Errorf("cannot find schema %s", entryName)
		return err
	}

	var obj interface{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	if err = schema.ValidateInterface(obj); err != nil {
		log.Info().Err(err).Msg(semLogContext)
		schemaErr, _ := NewSchemaErrorFromError(schema, err)
		return schemaErr
	}

	return nil
}
