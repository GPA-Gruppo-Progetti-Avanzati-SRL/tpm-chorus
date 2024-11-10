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

type Paths []Path

func (ps Paths) FindOutgoingPaths(activity string) Paths {

	var outPaths Paths
	for _, p := range ps {
		if p.SourceName == activity {
			outPaths = append(outPaths, p)
		}
	}

	return outPaths
}

func (ps Paths) FindIncomingPaths(activity string) Paths {

	var inPaths Paths
	for _, p := range ps {
		if p.TargetName == activity {
			inPaths = append(inPaths, p)
		}
	}

	return inPaths
}

func (ps Paths) RemoveActivityOutgoingPaths(activity string) Paths {

	var paths Paths
	for _, p := range ps {
		if p.SourceName != activity {
			paths = append(paths, p)
		}
	}

	return paths
}
