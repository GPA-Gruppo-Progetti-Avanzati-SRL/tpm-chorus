package kzxform

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

const (
	OperatorShift = "shift"
)

type TransformReference struct {
	Typ           string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty" json:"type,omitempty"`
	Id            string `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	DefinitionRef string `yaml:"definition-ref,omitempty" mapstructure:"definition-ref,omitempty" json:"definition-ref,omitempty"`
	Guard         string `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
	Data          []byte `yaml:"-" mapstructure:"-" json:"-"`
}

type Specs map[string]interface{}

type Rule struct {
	Operation     string `yaml:"operation,omitempty" mapstructure:"operation,omitempty" json:"operation,omitempty"`
	Specification Specs  `yaml:"spec,omitempty" mapstructure:"spec,omitempty" json:"spec,omitempty"`
	InPlace       bool   `yaml:"inplace,omitempty" mapstructure:"inplace,omitempty" json:"inplace,omitempty"`
	Require       bool   `yaml:"require,omitempty" mapstructure:"require,omitempty" json:"require,omitempty"`
	// SubRules      []Rule `yaml:"sub-rules,omitempty" mapstructure:"sub-rules,omitempty" json:"sub-rules,omitempty"`
}

type Config struct {
	Id      string `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	Verbose bool   `yaml:"verbose,omitempty" mapstructure:"verbose,omitempty" json:"verbose,omitempty"`
	Rules   []Rule `yaml:"rules,omitempty" mapstructure:"rules,omitempty" json:"rules,omitempty"`
}

func (d *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {

	type config Config

	t1 := config{}
	err := unmarshal(&t1)
	if err != nil {
		return err
	}

	d.Id = t1.Id
	d.Verbose = t1.Verbose
	d.Rules = t1.Rules
	for i, r := range t1.Rules {
		for specName, specValue := range r.Specification {
			if specName == "sub-rules" {
				subRules := make([]Rule, 0)
				err = mapstructure.Decode(specValue, &subRules)
				if err != nil {
					return err
				}
				for i, sr := range subRules {
					specValue := cleanUpSpecValue(sr.Specification)
					subRules[i].Specification = specValue
				}
				d.Rules[i].Specification[specName] = subRules
			} else {
				specValue = cleanUpValue(specValue)
				d.Rules[i].Specification[specName] = specValue
			}
		}
	}

	//type specs Specs
	//
	//tmp := make(specs)
	//err := unmarshal(tmp)
	//if err != nil {
	//	return err
	//}

	return nil
}

func cleanUpSpecValue(spec map[string]interface{}) map[string]interface{} {
	for n, v := range spec {
		v = cleanUpValue(v)
		spec[n] = v
	}

	return spec
}

func cleanUpValue(v interface{}) interface{} {

	if util.IsNilish(v) {
		return nil
	}

	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case Specs:
		return cleanUpStringMap(v)
	case map[string]interface{}:
		return cleanUpStringMap(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	case bool:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpValue(v)
	}
	return result
}

func cleanUpStringMap(in map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[k] = cleanUpValue(v)
	}
	return result
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpValue(v)
	}
	return result
}

func (t *Config) ToJSONRule() (string, error) {

	/*
		for i := 0; i < len(t.Rules); i++ {
			if len(t.Rules[i].SubRules) != 0 {
				t.Rules[i].Specification["sub-rules"] = t.Rules[i].SubRules
				t.Rules[i].SubRules = nil
			}
		}
	*/

	b, err := json.Marshal(t.Rules)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (t *Config) ToYamlRule() (string, error) {
	b, err := yaml.Marshal(t.Rules)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (t *Config) ToYaml() (string, error) {
	b, err := yaml.Marshal(t)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

/*
func (t *Config) GetRule() (string, error) {
	var sb strings.Builder
	sb.WriteString("[")
	for i, r := range t.Rules {
		if i > 0 {
			sb.WriteString(",")
		}
		b, err := json.Marshal(r)
		if err != nil {
			return "", err
		}
		sb.Write(b)
	}
	sb.WriteString("]")
	return sb.String(), nil
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	// Trim the beginning and trailing " character
	return string(b[1 : len(b)-1])
}
*/
