package xforms

type TransformReference struct {
	Typ           string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty" json:"type,omitempty"`
	Id            string `yaml:"id,omitempty" mapstructure:"id,omitempty" json:"id,omitempty"`
	DefinitionRef string `yaml:"definition-ref,omitempty" mapstructure:"definition-ref,omitempty" json:"definition-ref,omitempty"`
	Guard         string `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
	Data          []byte `yaml:"-" mapstructure:"-" json:"-"`
}
