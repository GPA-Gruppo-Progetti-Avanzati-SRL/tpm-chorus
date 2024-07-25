package repo

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"path/filepath"
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
	MountPoint string `yaml:"mount-point,omitempty" json:"mount-point,omitempty" mapstructure:"mount-point,omitempty"`
	Asset      Asset  `yaml:"root-asset,omitempty" json:"root-asset,omitempty" mapstructure:"root-asset,omitempty"`
	Refs       Assets `yaml:"assets,omitempty" json:"assets,omitempty" mapstructure:"assets,omitempty"`
}

var ZeroAssetGroup = AssetGroup{}

func (g AssetGroup) ReadRefsData(resource string) ([]byte, error) {
	const semLogContext = "asset-group::read-data"
	var err error

	ndx := g.Refs.IndexByPath(resource)
	if ndx < 0 {
		err = errors.New("cannot find schema file")
		log.Error().Err(err).Str("resource-file", resource).Msg(semLogContext)
		return nil, err
	}

	b, err := g.Refs[ndx].ReadData(g.MountPoint)
	if err != nil {
		log.Error().Err(err).Str("resource-file", resource).Msg(semLogContext)
		return nil, err
	}

	return b, nil
}

func (g AssetGroup) AssetIndexByPath(p string) int {
	return g.Refs.IndexByPath(p)
}

func (g AssetGroup) FindAssetIndexByPath(p string) int {
	//for i, a := range g.Refs {
	//	if a.Path == p {
	//		return i
	//	}
	//}

	return g.Refs.IndexByPath(p)
}

func (g AssetGroup) FindAssetIndexByType(t string) int {
	//for i, a := range g.Refs {
	//	if a.Type == t {
	//		return i
	//	}
	//}

	return g.Refs.IndexByType(t)
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

func (a Asset) ReadData(mountPoint string) ([]byte, error) {
	resolvedPath := filepath.Join(mountPoint, a.Path)
	b, err := util.ReadFileAndResolveEnvVars(resolvedPath)
	if err != nil {
		return nil, err
	}

	return b, err
}

type Assets []Asset

func (as Assets) IndexByPath(p string) int {
	for i, a := range as {
		if a.Path == p {
			return i
		}
	}

	return -1
}

func (as Assets) IndexByType(t string) int {
	for i, a := range as {
		if a.Type == t {
			return i
		}
	}

	return -1
}

type OrchestrationBundle struct {
	Path          string                `yaml:"path,omitempty" mapstructure:"path,omitempty"`
	Version       string                `yaml:"version,omitempty" mapstructure:"version,omitempty"`
	SHA           string                `yaml:"sha,omitempty" mapstructure:"sha,omitempty"`
	AssetGroup    AssetGroup            `yaml:"asset-group,omitempty" mapstructure:"asset-group,omitempty"`
	NestedBundles []OrchestrationBundle `yaml:"nested-bundles,omitempty" mapstructure:"nested-bundles,omitempty"`
}

func (r *OrchestrationBundle) ShowInfo() {
	const semLogContext = "orchestration-bundle::show-info"
	log.Info().Str(SemLogPath, r.Path).Msg(semLogContext)

	log.Info().Str("path", r.Path).Msg(semLogContext)
	log.Info().Str(SemLogType, r.AssetGroup.Asset.Type).Str(SemLogFile, r.AssetGroup.Asset.Path).Msg(semLogContext)
	for _, a := range r.AssetGroup.Refs {
		log.Info().Str(SemLogType, a.Type).Str(SemLogPath, r.Path).Str(SemLogFile, a.Path).Msg(semLogContext)
	}
}
