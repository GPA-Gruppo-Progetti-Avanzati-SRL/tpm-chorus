package repo_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"testing"
)

const OrchestrationFolder = "../../../examples/open-api-repo-example-01/orchestration1"

func TestLoadOrchestrationRepo(t *testing.T) {

	const semLogContext = "examples::test-load-orchestration-repo"

	sarr := []string{
		OrchestrationFolder,
	}

	for _, dir := range sarr {
		bundle, err := repo.LoadOrchestrationRepo(dir)
		require.NoError(t, err)

		log.Info().Str("fld", bundle.Path).Msg(semLogContext)
		log.Info().
			Str("asset-root-path", bundle.AssetGroup.Root.Path).
			Str("asset-root-name", bundle.AssetGroup.Root.Name).
			Str("asset-root-type", bundle.AssetGroup.Root.Type).Msg(semLogContext)
		for _, a := range bundle.AssetGroup.Refs {
			log.Info().Str("type", a.Type).Str("asset-path", a.Path).Str("asset-name", a.Name).Msg(semLogContext)
		}

		_, _, err = bundle.LoadOrchestrationData()
		require.NoError(t, err)
	}

}
