package configBundle

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
	Path string `yaml:"path,omitempt" mapstructure:"path,omitempt"`
	//Version     string                `yaml:"version,omitempt" mapstructure:"version,omitempty"`
	//SHA         string                `yaml:"sha,omitempt" mapstructure:"sha,omitempt"`
	AssetGroups map[string]AssetGroup `yaml:"asset-groups,omitempt" mapstructure:"asset-groups,omitempt"`
}

func (r *OrchestrationBundle) ShowInfo() {
	log.Info().Str(SemLogPath, r.GetPath()).Msg(SemLogRepo)

	for no, o := range r.AssetGroups {
		log.Info().Str(SemLogOrchestrationSid, no).Msg("orchestration")
		log.Info().Str(SemLogType, o.Root.Type).Str(SemLogPath, r.GetPath()).Str(SemLogFile, o.Root.Path).Msg("orchestration main")
		for _, a := range o.Refs {
			log.Info().Str(SemLogType, a.Type).Str(SemLogPath, r.GetPath()).Str(SemLogFile, a.Path).Msg("orchestration asset")
		}
	}
}
