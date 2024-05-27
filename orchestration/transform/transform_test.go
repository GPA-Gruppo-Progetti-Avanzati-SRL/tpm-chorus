package transform_test

import (
	_ "embed"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"io/fs"
	"os"
	"testing"
	"tpm-chorus/orchestration/transform"
)

func handleErrorTestMain(err error) {
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	err := transform.InitializeKazaamRegistry()
	handleErrorTestMain(err)

	registry := transform.GetRegistry()

	trsf1 := transform.Config{}
	err = yaml.Unmarshal(case001RuleYml, &trsf1)
	handleErrorTestMain(err)
	err = registry.Add(trsf1)
	handleErrorTestMain(err)

	trsf2 := transform.Config{}
	err = yaml.Unmarshal(case002RuleYml, &trsf2)
	handleErrorTestMain(err)
	err = registry.Add(trsf2)
	handleErrorTestMain(err)

	trsf3 := transform.Config{}
	err = yaml.Unmarshal(case003RuleYml, &trsf3)
	handleErrorTestMain(err)
	err = registry.Add(trsf3)
	handleErrorTestMain(err)

	trsf3b := transform.Config{}
	err = yaml.Unmarshal(case003bRuleYml, &trsf3b)
	handleErrorTestMain(err)
	err = registry.Add(trsf3b)
	handleErrorTestMain(err)

	trsf4 := transform.Config{}
	err = yaml.Unmarshal(case004RuleYml, &trsf4)
	handleErrorTestMain(err)
	err = registry.Add(trsf4)
	handleErrorTestMain(err)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestOperators(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	registry := transform.GetRegistry()

	var trsf transform.Transformation
	var err error
	var dataOut []byte
	trsf, err = registry.Get("case003")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case003", []byte(case003Input))
	require.NoError(t, err)
	err = os.WriteFile("case-003-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)

	trsf, err = registry.Get("case003b")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case003b", []byte(case003bInput))
	require.NoError(t, err)
	err = os.WriteFile("case-003b-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)

	trsf, err = registry.Get("case001")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case001", []byte(case001Input))
	require.NoError(t, err)
	err = os.WriteFile("case-001-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)

	trsf, err = registry.Get("case002")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case002", []byte(case002Input))
	require.NoError(t, err)
	err = os.WriteFile("case-002-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)

	trsf, err = registry.Get("case004")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case004", []byte(case004Input))
	require.NoError(t, err)
	err = os.WriteFile("case-004-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)
}

func BenchmarkOperators(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	registry := transform.GetRegistry()

	for i := 0; i < b.N; i++ {
		_, err := registry.Get("case004")
		require.NoError(b, err)

		_, err = registry.Transform("case004", []byte(case004Input))
		require.NoError(b, err)
	}
}
