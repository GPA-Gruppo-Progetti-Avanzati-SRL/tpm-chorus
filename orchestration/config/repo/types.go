package repo

import (
	"github.com/rs/zerolog/log"
)

const (
	SemLogRepo = "repo"

	SemLogOrchestrationSid = "sid"
	SemLogPath             = "path"
	SemLogFile             = "file"
	SemLogType             = "type"
	SemLogName             = "name"
	SemLogMethod           = "method"

	SemLogActivity             = "act"
	SemLogNextActivity         = "next-act"
	SemLogActivityName         = "name"
	SemLogActivityNumOutputs   = "len-outputs"
	SemLogActivityNumInputs    = "len-inputs"
	SemLogCacheKey             = "cache-key"
	ContentTypeApplicationJson = "application/json"

	VERSION string = "0.0.1-SNAPSHOT"
)

type AssetGroup struct {
	Root Asset
	Refs []Asset `yaml:"references" json:"data" mapstructure:"references"`
}

func (g AssetGroup) FindAssetIndexByPath(p string) int {
	for i, a := range g.Refs {
		if a.Path == p {
			return i
		}
	}

	return -1
}

func (g AssetGroup) FindAssetIndexByType(t string) int {
	for i, a := range g.Refs {
		if a.Type == t {
			return i
		}
	}

	return -1
}

type Asset struct {
	Name string `yaml:"name" json:"name" mapstructure:"name"`
	Type string `yaml:"type" json:"type" mapstructure:"type"`
	Path string `yaml:"path" json:"path" mapstructure:"path"`
	Data []byte `yaml:"data" json:"data" mapstructure:"data"`
}

func (a Asset) IsZero() bool {
	return a.Type == ""
}

type OrchestrationBundle struct {
	Path       string     `yaml:"path,omitempty" mapstructure:"path,omitempty"`
	Version    string     `yaml:"version,omitempty" mapstructure:"version,omitempty"`
	SHA        string     `yaml:"sha,omitempty" mapstructure:"sha,omitempty"`
	AssetGroup AssetGroup `yaml:"asset-group,omitempty" mapstructure:"asset-group,omitempty"`
}

func (r *OrchestrationBundle) ShowInfo() {
	const semLogContext = "orchestration-bundle::show-info"
	log.Info().Str(SemLogPath, r.Path).Msg(semLogContext)

	log.Info().Str("path", r.Path).Msg(semLogContext)
	log.Info().Str(SemLogType, r.AssetGroup.Root.Type).Str(SemLogFile, r.AssetGroup.Root.Path).Msg(semLogContext)
	for _, a := range r.AssetGroup.Refs {
		log.Info().Str(SemLogType, a.Type).Str(SemLogPath, r.Path).Str(SemLogFile, a.Path).Msg(semLogContext)
	}
}
