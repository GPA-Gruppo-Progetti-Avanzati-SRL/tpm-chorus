package kzxform_test

import (
	_ "embed"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
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
	err := kzxform.InitializeKazaamRegistry()
	handleErrorTestMain(err)

	registry := kzxform.GetRegistry()

	trsf0 := kzxform.Config{}
	err = yaml.Unmarshal(t1, &trsf0)
	handleErrorTestMain(err)

	trsf1 := kzxform.Config{}
	err = yaml.Unmarshal(case001RuleYml, &trsf1)
	handleErrorTestMain(err)
	err = registry.Add3(trsf1)
	handleErrorTestMain(err)

	trsf2 := kzxform.Config{}
	err = yaml.Unmarshal(case002RuleYml, &trsf2)
	handleErrorTestMain(err)
	err = registry.Add3(trsf2)
	handleErrorTestMain(err)

	trsf3 := kzxform.Config{}
	err = yaml.Unmarshal(case003RuleYml, &trsf3)
	handleErrorTestMain(err)
	err = registry.Add3(trsf3)
	handleErrorTestMain(err)

	trsf3b := kzxform.Config{}
	err = yaml.Unmarshal(case003bRuleYml, &trsf3b)
	handleErrorTestMain(err)
	err = registry.Add3(trsf3b)
	handleErrorTestMain(err)

	trsf4 := kzxform.Config{}
	err = yaml.Unmarshal(case004RuleYml, &trsf4)
	handleErrorTestMain(err)
	err = registry.Add3(trsf4)
	handleErrorTestMain(err)

	trsf5 := kzxform.Config{}
	err = yaml.Unmarshal(case005RuleYml, &trsf5)
	handleErrorTestMain(err)
	err = registry.Add3(trsf5)
	handleErrorTestMain(err)

	trsf6 := kzxform.Config{}
	err = yaml.Unmarshal(case006RuleYml, &trsf6)
	handleErrorTestMain(err)
	err = registry.Add3(trsf6)
	handleErrorTestMain(err)

	trsf7 := kzxform.Config{}
	err = yaml.Unmarshal(case007RuleYml, &trsf7)
	handleErrorTestMain(err)
	err = registry.Add3(trsf7)
	handleErrorTestMain(err)

	trsf8 := kzxform.Config{}
	err = yaml.Unmarshal(case008RuleYml, &trsf8)
	handleErrorTestMain(err)
	err = registry.Add3(trsf8)
	handleErrorTestMain(err)

	trsf9 := kzxform.Config{}
	err = yaml.Unmarshal(case009RuleYml, &trsf9)
	handleErrorTestMain(err)
	err = registry.Add3(trsf9)
	handleErrorTestMain(err)

	trsf10 := kzxform.Config{}
	err = yaml.Unmarshal(case010RuleYml, &trsf10)
	handleErrorTestMain(err)
	err = registry.Add3(trsf10)
	handleErrorTestMain(err)

	trsf11 := kzxform.Config{}
	err = yaml.Unmarshal(case011RuleYml, &trsf11)
	handleErrorTestMain(err)
	err = registry.Add3(trsf11)
	handleErrorTestMain(err)

	trsf12 := kzxform.Config{}
	err = yaml.Unmarshal(case012RuleYml, &trsf12)
	handleErrorTestMain(err)
	err = registry.Add3(trsf12)
	handleErrorTestMain(err)

	trsf13 := kzxform.Config{}
	err = yaml.Unmarshal(case013RuleYml, &trsf13)
	handleErrorTestMain(err)
	err = registry.Add3(trsf13)
	handleErrorTestMain(err)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestSingleCase(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	registry := kzxform.GetRegistry()

	var trsf kzxform.Transformation
	var err error
	var dataOut []byte

	rule := "case013"
	input := case013Input
	output := "case-013-output.json"
	trsf, err = registry.Get(rule)
	require.NoError(t, err)
	t.Log(trsf.Cfg.ToYaml())
	dataOut, err = registry.Transform(rule, []byte(input))
	require.NoError(t, err)
	err = os.WriteFile(output, dataOut, fs.ModePerm)
	require.NoError(t, err)
}

func TestOperators(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	registry := kzxform.GetRegistry()

	var trsf kzxform.Transformation
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

	registry := kzxform.GetRegistry()

	for i := 0; i < b.N; i++ {
		_, err := registry.Get("case004")
		require.NoError(b, err)

		_, err = registry.Transform("case004", []byte(case004Input))
		require.NoError(b, err)
	}
}
