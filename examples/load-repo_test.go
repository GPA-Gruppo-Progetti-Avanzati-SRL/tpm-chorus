package examples_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry/configBundle"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	CrawlerMountPoint   = "./open-api-repo-example-01"
	OrchestrationFolder = "./open-api-repo-example-01/orchestration1"
)

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

func TestLoadOrchestrationRepo(t *testing.T) {

	const semLogContext = "examples::test-load-orchestration-repo"

	sarr := []string{
		OrchestrationFolder,
	}

	for _, dir := range sarr {
		orchestrationFld, assetGroup, err := configBundle.LoadOrchestrationRepo(dir)
		require.NoError(t, err)

		log.Info().Str("fld", orchestrationFld).Str("type", assetGroup.Root.Type).Msg(semLogContext)

		for _, a := range assetGroup.Refs {
			log.Info().Str("type", a.Type).Str("asset-path", a.Path).Msg(semLogContext)
		}

		_, _, err = configBundle.LoadOrchestrationData(CrawlerMountPoint, assetGroup)
		require.NoError(t, err)
	}

}
