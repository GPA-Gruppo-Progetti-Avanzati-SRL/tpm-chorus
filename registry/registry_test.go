package registry_test

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"tpm-chorus/registry"
)

func TestLoadRegistry(t *testing.T) {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	path, err := os.Getwd()
	require.NoError(t, err)
	t.Logf("current directory: %s", path)

	sarr := []string{
		"../examples",
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
