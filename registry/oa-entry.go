package registry

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
	"tpm-chorus/constants"
	"tpm-chorus/orchestration/config"
	"tpm-chorus/registry/configBundle"
)

type OpenApiRegistryEntry struct {
	Path                string `yaml:"path,omitempt" mapstructure:"path,omitempt"`
	Version             string `yaml:"version,omitempt" mapstructure:"version,omitempty"`
	SHA                 string `yaml:"sha,omitempt" mapstructure:"sha,omitempt"`
	OrchestrationBundle configBundle.OrchestrationBundle
	OpenApiBundle       configBundle.AssetGroup `yaml:"api" mapstructure:"api"`
	Orchestrations      []config.Orchestration
	OpenapiDoc          *openapi3.T
}

func (r *OpenApiRegistryEntry) ShowInfo() {
	log.Info().Str(constants.SemLogPath, r.OrchestrationBundle.GetPath()).Msg("BOF OpenApi repo information --------")
	log.Info().Str(constants.SemLogPath, r.OrchestrationBundle.GetPath()).Str(constants.SemLogFile, r.OpenApiBundle.Root.Path).Msg(constants.SemLogOpenApi)
	for _, a := range r.OpenApiBundle.Refs {
		log.Info().Str(constants.SemLogType, a.Type).Str(constants.SemLogPath, r.OrchestrationBundle.GetPath()).Str(constants.SemLogFile, a.Path).Msg("open-api external value")
	}

	r.OrchestrationBundle.ShowInfo()
}

func (r *OpenApiRegistryEntry) getOpenApiData() (string, []byte, error) {
	if r.OpenApiBundle.Root.IsZero() {
		return "", nil, fmt.Errorf("no api definition found in repo %s", r.Path)
	}

	if r.OpenApiBundle.Root.Data != nil {
		return "", r.OpenApiBundle.Root.Data, nil
	}

	resolvedPath := filepath.Join(r.Path, r.OpenApiBundle.Root.Path)
	b, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", nil, err
	}

	r.OpenApiBundle.Root.Data = b
	return r.OpenApiBundle.Root.Path, r.OpenApiBundle.Root.Data, nil
}

func (r *OpenApiRegistryEntry) GetRefAssetData(fn string) ([]byte, error) {

	refs := r.OpenApiBundle.Refs
	if len(refs) == 0 {
		return nil, fmt.Errorf("no open-api assets references found in repo %s", r.Path)
	}

	ndx := configBundle.FindAssetIndexByPath(refs, fn)
	if ndx < 0 {
		return nil, fmt.Errorf("no open-api assets references found in repo %s for %s", r.Path, fn)
	}

	if refs[ndx].Data != nil {
		return refs[ndx].Data, nil
	}

	resolvedPath := filepath.Join(r.Path, refs[ndx].Path)
	b, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	refs[ndx].Data = b
	return b, nil
}

func (r *OpenApiRegistryEntry) GetOpenApiVersionAndSha() (string, string) {

	const semLogContext = "orchestration-repo::get-openapi-version-and-sha"

	var version, sha string

	if r.OpenApiBundle.Root.IsZero() {
		return version, sha
	}

	ndx := r.OpenApiBundle.FindAssetIndexByType(configBundle.AssetTypeVersion)
	if ndx >= 0 {
		b, err := r.GetRefAssetData(r.OpenApiBundle.Refs[ndx].Path)
		if err == nil {
			version = strings.TrimSpace(string(b))
		}
	}

	ndx = r.OpenApiBundle.FindAssetIndexByType(configBundle.AssetTypeSHA)
	if ndx >= 0 {
		b, err := r.GetRefAssetData(r.OpenApiBundle.Refs[ndx].Path)
		if err == nil {
			sha = strings.TrimSpace(string(b))
		}
	}

	if version != "" || sha != "" {
		log.Trace().Str("version", version).Str("SHA", sha).Msg(semLogContext)
	}

	return version, sha
}
