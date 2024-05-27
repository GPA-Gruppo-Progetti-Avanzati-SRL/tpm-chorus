package config

type Path struct {
	SourceName string `yaml:"source,omitempty" mapstructure:"source,omitempty" json:"source,omitempty"`
	TargetName string `yaml:"target,omitempty" mapstructure:"target,omitempty" json:"target,omitempty"`
	Constraint string `yaml:"constraint,omitempty" mapstructure:"constraint,omitempty" json:"constraint,omitempty"`
}

func NewPath(source string, target string, constraint string) *Path {
	p := Path{SourceName: source, TargetName: target, Constraint: constraint}
	return &p
}
