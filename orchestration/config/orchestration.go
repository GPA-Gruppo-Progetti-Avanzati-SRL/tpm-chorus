package config

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry/configBundle"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"strings"
)

/*
 * DataReferences
 */

type DataReference struct {
	Path string `yaml:"ref-path,omitempty" mapstructure:"ref-path,omitempty" json:"ref-path,omitempty"`
	Data []byte `yaml:"-" mapstructure:"-" json:"-"`
}

type DataReferences []DataReference

func (dr DataReferences) Find(p string) ([]byte, bool) {
	for _, r := range dr {
		if r.Path == p {
			return r.Data, true
		}
	}

	return nil, false
}

func (dr DataReferences) IsPresent(p string) bool {
	for _, r := range dr {
		if r.Path == p {
			return true
		}
	}

	return false
}

/*
 * ProcessVariables
 */

//type ProcessVarDefinitionValueExpressionTerm struct {
//	TermType  int
//	TermParam string
//}
//
//type ProcessVarDefinitionValueExpression struct {
//	Terms []ProcessVarDefinitionValueExpressionTerm
//}
//
//func (expr *ProcessVarDefinitionValueExpression) IsEmpty() bool {
//	return len(expr.Terms) == 0
//}

type ProcessVar struct {
	Name  string `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Value string `yaml:"value,omitempty" mapstructure:"value,omitempty" json:"value,omitempty"`
	Type  string `yaml:"type,omitempty" mapstructure:"type,omitempty" json:"type,omitempty"`
	Guard string `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
	// ParsedExpr ProcessVarDefinitionValueExpression `mapstructure:"-" yaml:"-" json:"-"`
}

/*
func ParseProcessVarDefinitionValueExpression(v string) (ProcessVarDefinitionValueExpression, error) {

	expr := ProcessVarDefinitionValueExpression{Terms: nil}
	vals := make([]ProcessVarDefinitionValueExpressionTerm, 0)

	// Get the vars defined.
	vars, err := varResolver.FindVariableReferences(v, varResolver.AnyVariableReference)
	if err != nil {
		return ProcessVarDefinitionValueExpression{}, err
	}

	// replace the vars with a comma....
	tmpV, _ := varResolver.ResolveVariables(v, varResolver.AnyVariableReference, func(s string) string { return "{var}," })

	// Now split with commas....
	varsNdx := 0
	sarr := strings.Split(tmpV, ",")
	for i, s := range sarr {
		if s == "{var}" {
			s = vars[varsNdx].VarName
			varsNdx++
		}

		if s != "" {
			fmt.Printf("%d = %s\n", i, s)
		}
	}

	expr.Terms = vals
	return expr, nil
}
*/

type ProcessVars struct {
	Vars        []ProcessVar `yaml:"vars,omitempty" mapstructure:"vars,omitempty" json:"vars,omitempty"`
	Validations []string     `yaml:"validations,omitempty" mapstructure:"validations,omitempty" json:"validations,omitempty"`
}

type ExecBoundary struct {
	Name       string   `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Activities []string `yaml:"activities,omitempty" mapstructure:"activities,omitempty" json:"activities,omitempty"`
}

/*
 * Orchestration
 */

type Orchestration struct {
	Id            string                            `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	Description   string                            `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	Version       string                            `yaml:"version,omitempty" mapstructure:"version,omitempty" json:"version,omitempty"`
	SHA           string                            `yaml:"sha,omitempty" mapstructure:"sha,omitempty" json:"sha,omitempty"`
	StartActivity string                            `json:"-" yaml:"-"`
	Paths         []Path                            `yaml:"paths,omitempty" mapstructure:"paths,omitempty" json:"paths,omitempty"`
	Activities    []Configurable                    `json:"-" yaml:"activities"`
	Boundaries    []ExecBoundary                    `yaml:"boundaries,omitempty" mapstructure:"boundaries,omitempty" json:"boundaries,omitempty"`
	RawActivities []json.RawMessage                 `json:"activities" yaml:"-"`
	References    DataReferences                    `yaml:"-" mapstructure:"-" json:"-"`
	Dictionaries  Dictionaries                      `yaml:"-" mapstructure:"-" json:"-"`
	PII           PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
}

func NewOrchestrationDefinitionFromFolder(orchestration1Folder string) (Orchestration, error) {
	const semLogContext = "config::new-orchestration-definition-from-folder"
	bundle, err := repo.NewOrchestrationBundleFromFolder(orchestration1Folder)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return Orchestration{}, err
	}

	orchestrationDefinition, err := NewOrchestrationDefinitionFromBundle(&bundle)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	return orchestrationDefinition, err
}

func NewOrchestrationDefinitionFromBundle(bundle *repo.OrchestrationBundle) (Orchestration, error) {

	orchestrationDefinitionData, assets, err := bundle.LoadOrchestrationData()
	if err != nil {
		return Orchestration{}, err
	}

	o, err := NewOrchestrationFromYAML(orchestrationDefinitionData)
	if err != nil {
		return Orchestration{}, err
	}

	for _, a := range assets {
		switch a.Type {
		case configBundle.AssetTypeDictionary:
			var d Dictionary
			d, err = NewDictionary(a.Name, a.Data)
			o.Dictionaries = append(o.Dictionaries, d)
		case configBundle.AssetTypeSHA:
			o.SHA = strings.TrimSpace(string(a.Data))
		case configBundle.AssetTypeVersion:
			o.Version = strings.TrimSpace(string(a.Data))
		default:
			o.References = append(o.References, DataReference{Path: a.Name, Data: a.Data})
		}

		if err != nil {
			return Orchestration{}, err
		}
	}

	return o, nil
}

func NewOrchestrationFromJSON(data []byte) (Orchestration, error) {
	o := Orchestration{}
	err := json.Unmarshal(data, &o)

	o.PII.Initialize()
	return o, err
}

func NewOrchestrationFromYAML(data []byte) (Orchestration, error) {
	o := Orchestration{}
	err := yaml.Unmarshal(data, &o)

	o.PII.Initialize()
	return o, err
}

func (o *Orchestration) ToJSON() ([]byte, error) {
	return json.Marshal(o)
}

func (o *Orchestration) ToYAML() ([]byte, error) {
	return yaml.Marshal(o)
}

func (o *Orchestration) FindActivityByName(n string) Configurable {
	for _, a := range o.Activities {
		if a.Name() == n {
			return a
		}
	}

	return nil
}

func (o *Orchestration) FindBoundaryByName(n string) (ExecBoundary, bool) {
	for _, a := range o.Boundaries {
		if a.Name == n {
			return a, true
		}
	}

	return ExecBoundary{}, false
}

func (o *Orchestration) AddActivity(a Configurable) error {

	if o.FindActivityByName(a.Name()) != nil {
		return fmt.Errorf("activity with the same id already present (id: %s)", a.Name())
	}

	if a.Type() == RequestActivityType && o.StartActivity != "" {
		return fmt.Errorf("dup start activity (current: %s, dup: %s)", o.StartActivity, a.Name())
	} else {
		o.StartActivity = a.Name()
	}

	o.Activities = append(o.Activities, a)
	return nil
}

func (o *Orchestration) AddPath(source, target, constraint string) error {

	if source == "" || target == "" {
		return fmt.Errorf("path missing source or target reference")
	}

	if o.FindActivityByName(source) == nil {
		return fmt.Errorf("cannot find source activity (id: %s)", source)
	}

	if o.FindActivityByName(target) == nil {
		return fmt.Errorf("cannot find target activity (id: %s)", target)
	}

	o.Paths = append(o.Paths, Path{SourceName: source, TargetName: target, Constraint: constraint})
	return nil
}

func (o *Orchestration) UnmarshalJSON(b []byte) error {

	// Clear the state....
	o.Activities = nil

	type orchestration Orchestration
	err := json.Unmarshal(b, (*orchestration)(o))
	if err != nil {
		return err
	}

	for _, raw := range o.RawActivities {
		var v Activity
		err = json.Unmarshal(raw, &v)
		if err != nil {
			return err
		}
		/*
			var i Configurable
			switch v.Type() {
			case RequestActivityType:
				i = NewRequestActivity()
			case EchoActivityType:
				i = NewEchoActivity()
			case ResponseActivityType:
				i = NewResponseActivity()
			default:
				return fmt.Errorf("unknown activity type %s", v.Type())
			}err = json.Unmarshal(raw, i)
			if err != nil {
				return err
			}
		*/

		i, err := NewActivityFromJSON(v.Type(), raw)
		if err != nil {
			return err
		}

		o.AddActivity(i)
	}
	return nil
}

func (o *Orchestration) MarshalJSON() ([]byte, error) {

	// Clear the state....
	o.RawActivities = nil

	type orchestration Orchestration
	if o.Activities != nil {
		for _, v := range o.Activities {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			o.RawActivities = append(o.RawActivities, b)
		}
	}
	return json.Marshal((*orchestration)(o))
}

//func (o *Orchestration) MarshalYAMLw() (interface{}, error) {
//	type orchestration Orchestration
//	Clear the state....
//
//o.IntfActivities = nil
//
//
//if o.Activities != nil {
//	for _, v := range o.Activities {
//		/*
//			b, err := yaml.Marshal(v)
//			if err != nil {
//				return nil, err
//			}
//		*/
//		o.IntfActivities = append(o.IntfActivities, v)
//	}
//}
//
//	return yaml.Marshal((*orchestration)(o))
//}

func (o *Orchestration) UnmarshalYAML(unmarshal func(interface{}) error) error {

	type orchestration Orchestration

	var m struct {
		Id          string                            `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
		Description string                            `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
		Paths       []Path                            `yaml:"paths,omitempty" mapstructure:"paths,omitempty" json:"paths,omitempty"`
		Boundaries  []ExecBoundary                    `yaml:"boundaries,omitempty" mapstructure:"boundaries,omitempty" json:"boundaries,omitempty"`
		Activities  []interface{}                     `json:"activities" yaml:"activities"`
		PII         PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
	}
	m.Activities = make([]interface{}, 0)
	err := unmarshal(&m)
	if err != nil {
		return err
	}

	o.Paths = m.Paths
	o.Id = m.Id
	o.Description = m.Description
	o.Boundaries = m.Boundaries
	o.PII = m.PII
	for _, a := range m.Activities {
		var wa struct {
			Activity Activity
		}
		err := mapstructure.Decode(a, &wa)
		if err != nil {
			return err
		}

		i, err := NewActivityFromYAML(wa.Activity.Type(), a)
		if err != nil {
			return err
		}

		/*
			switch wa.Activity.Type() {
			case RequestActivityType:
				sa := RequestActivity{}
				err := mapstructure.Decode(a, &sa)
				if err != nil {
					return err
				}
			case ResponseActivityType:
			case EchoActivityType:
			}
		*/

		o.Activities = append(o.Activities, i)
	}

	return nil
}
