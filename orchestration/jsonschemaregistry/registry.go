package jsonschemaregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/santhosh-tekuri/jsonschema"
	"strings"
)

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
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}
