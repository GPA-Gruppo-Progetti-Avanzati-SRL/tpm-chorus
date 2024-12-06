package transform_test

import (
	_ "embed"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"testing"
)

func handleErrorTestMain(err error) {
	if err != nil {
		panic(err)
	}
}

var t1 = []byte(`
id: "transform_activity_get_movie_kazaam_rule"
rules:
  - operation: shift
    spec:
      properties:
        cast: cast
        awards: awards
        fullplot: fullplot
        title: title
        year: year
`)

func TestMain(m *testing.M) {
	err := transform.InitializeKazaamRegistry()
	handleErrorTestMain(err)

	registry := transform.GetRegistry()

	trsf0 := transform.Config{}
	err = yaml.Unmarshal(t1, &trsf0)
	handleErrorTestMain(err)

	trsf1 := transform.Config{}
	err = yaml.Unmarshal(case001RuleYml, &trsf1)
	handleErrorTestMain(err)
	err = registry.Add3(trsf1)
	handleErrorTestMain(err)

	trsf2 := transform.Config{}
	err = yaml.Unmarshal(case002RuleYml, &trsf2)
	handleErrorTestMain(err)
	err = registry.Add3(trsf2)
	handleErrorTestMain(err)

	trsf3 := transform.Config{}
	err = yaml.Unmarshal(case003RuleYml, &trsf3)
	handleErrorTestMain(err)
	err = registry.Add3(trsf3)
	handleErrorTestMain(err)

	trsf3b := transform.Config{}
	err = yaml.Unmarshal(case003bRuleYml, &trsf3b)
	handleErrorTestMain(err)
	err = registry.Add3(trsf3b)
	handleErrorTestMain(err)

	trsf4 := transform.Config{}
	err = yaml.Unmarshal(case004RuleYml, &trsf4)
	handleErrorTestMain(err)
	err = registry.Add3(trsf4)
	handleErrorTestMain(err)

	trsf5 := transform.Config{}
	err = yaml.Unmarshal(case005RuleYml, &trsf5)
	handleErrorTestMain(err)
	err = registry.Add3(trsf5)
	handleErrorTestMain(err)

	trsf6 := transform.Config{}
	err = yaml.Unmarshal(case006RuleYml, &trsf6)
	handleErrorTestMain(err)
	err = registry.Add3(trsf6)
	handleErrorTestMain(err)

	trsf7 := transform.Config{}
	err = yaml.Unmarshal(case007RuleYml, &trsf7)
	handleErrorTestMain(err)
	err = registry.Add3(trsf7)
	handleErrorTestMain(err)

	trsf8 := transform.Config{}
	err = yaml.Unmarshal(case008RuleYml, &trsf8)
	handleErrorTestMain(err)
	err = registry.Add3(trsf8)
	handleErrorTestMain(err)

	trsf9 := transform.Config{}
	err = yaml.Unmarshal(case009RuleYml, &trsf9)
	handleErrorTestMain(err)
	err = registry.Add3(trsf9)
	handleErrorTestMain(err)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestSingleCase(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	registry := transform.GetRegistry()

	var trsf transform.Transformation
	var err error
	var dataOut []byte

	trsf, err = registry.Get("case009")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case009", []byte(case009Input))
	require.NoError(t, err)
	err = os.WriteFile("case-009-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)
}

func TestOperators(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	registry := transform.GetRegistry()

	var trsf transform.Transformation
	var err error
	var dataOut []byte

	trsf, err = registry.Get("case005")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case005", []byte(case005Input))
	require.NoError(t, err)
	err = os.WriteFile("case-005-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)

	trsf, err = registry.Get("case006")
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform("case006", []byte(case006Input))
	require.NoError(t, err)
	err = os.WriteFile("case-006-output.json", dataOut, fs.ModePerm)
	require.NoError(t, err)

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
