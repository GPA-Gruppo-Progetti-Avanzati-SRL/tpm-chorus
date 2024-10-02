package examples_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	// OrchestrationFolder = "./movies-orchestration"
	OrchestrationFolder = "../examples/test2"
)

/*
func TestLoadOrchestrationRepo1(t *testing.T) {

	sarr := []string{
		CrawlerMountPoint,
	}

	for _, dir := range sarr {
		cfg := &registry.Config{
			MountPoint: dir,
		}

		reg, err := registry.LoadRegistry(cfg)
		require.NoError(t, err)

		reg.ShowInfo()
	}
}
*/

func TestLoadOrchestrationRepo(t *testing.T) {

	const semLogContext = "examples::test-load-orchestration-repo"

	sarr := []string{
		OrchestrationFolder,
	}

	for _, dir := range sarr {
		bundle, err := repo.NewOrchestrationBundleFromFolder(dir)
		require.NoError(t, err)

		log.Info().Str("fld", bundle.Path).Str("type", bundle.AssetGroup.Asset.Type).Msg(semLogContext)

		for _, a := range bundle.AssetGroup.Refs {
			log.Info().Str("type", a.Type).Str("asset-path", a.Path).Msg(semLogContext)
		}

		_, _, err = bundle.LoadOrchestrationData()
		require.NoError(t, err)
	}

}
