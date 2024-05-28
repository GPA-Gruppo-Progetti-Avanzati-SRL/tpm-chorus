package registry

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry/configBundle"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	MountPoint string `yaml:"mount-point" mapstructure:"mount-point" json:"mount-point"`
}

func Crawl(cfg *Config) ([]OpenApiRegistryEntry, error) {
	fi, err := scan4OpenApiFiles(cfg.MountPoint)
	if err != nil {
		return nil, err
	}

	repos := make([]OpenApiRegistryEntry, 0)
	for _, f := range fi {

		assets, err := configBundle.Scan4AssetFiles(filepath.Dir(f))
		if err != nil {
			return repos, err
		}

		oapirepo := OpenApiRegistryEntry{
			Path:          filepath.Dir(f),
			OpenApiBundle: configBundle.AssetGroup{Root: configBundle.Asset{Name: filepath.Base(f), Type: configBundle.AssetTypeOpenAPi, Path: filepath.Base(f)}, Refs: assets},
		}
		oapirepo.OrchestrationBundle, err = configBundle.NewOrchestrationRepoFromFolder(filepath.Dir(f))
		if err != nil {
			return nil, err
		}
		repos = append(repos, oapirepo)
	}

	return repos, nil
}

func Scan4TopLevelVersionAndSHA(dir string) (string, string, error) {

	const semLogContext = "crawler::scan-4-top-level-version-sha"

	var version, sha string
	files, err := util.FindFiles(dir, util.WithFindOptionIncludeList(configBundle.VersionSHAFileFindIncludeList))
	if err != nil {
		return version, sha, err
	}

	for _, fn := range files {
		baseName := strings.ToUpper(filepath.Base(fn))
		b, err := os.ReadFile(fn)
		if err != nil {
			log.Error().Err(err).Str("fn", fn).Msg("error reading version file")
			return version, sha, err
		}

		switch baseName {
		case configBundle.SHAFileName:
			sha = string(b)
		case configBundle.VERSIONFileName:
			version = string(b)
		}
	}

	if version != "" || sha != "" {
		log.Trace().Str("version", version).Str("SHA", sha).Msg(semLogContext)
	}
	return version, sha, nil
}

func scan4OpenApiFiles(dir string) ([]string, error) {

	files, err := util.FindFiles(dir, util.WithFindOptionNavigateSubDirs(), util.WithFindOptionIncludeList(configBundle.OpenApiFileFindIncludeList))
	if err != nil {
		return nil, err
	}

	var openApiFiles []string
	for _, fn := range files {
		//baseName := strings.ToUpper(filepath.Base(fn))
		//if baseName == SHAFileName || baseName == VERSIONFileName {
		//	b, err := os.ReadFile(fn)
		//	if err != nil {
		//		log.Error().Err(err).Str("fn", fn).Msg("error reading version file")
		//	} else {
		//		log.Info().Str("fn", fn).Str("content", string(b)).Msg("found version file")
		//	}
		//} else {
		openApiFiles = append(openApiFiles, fn)
		//}
	}
	return openApiFiles, nil
}
