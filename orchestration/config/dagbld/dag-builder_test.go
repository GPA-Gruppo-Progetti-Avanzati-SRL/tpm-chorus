package dagbld_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/dagbld"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type TestModel struct {
}

func (t TestModel) AddNopActivity(d string) string {
	return util.NewUUID()
}

func (t TestModel) AddPath(src, target, condition string) error {
	log.Info().Str("src", src).Str("to", target).Str("if", condition).Msg("add-path")
	return nil
}

func TestDagBuilder(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	dag := dagbld.NewDAGPathBuilder(&TestModel{})
	dag.With(
		dag.S("start-activity"),
		dag.Switch(
			dag.Case("case1", dag.S(fmt.Sprintf("echo-activity-%d", 1))),
			dag.Case("case2", dag.S(fmt.Sprintf("echo-activity-%d", 2))),
		),
		dag.If("cond1",
			dag.S(fmt.Sprintf("echo-activity-%d", 3)),
			dag.Block(
				dag.S(fmt.Sprintf("echo-activity-%d", 4)),
				dag.S(fmt.Sprintf("echo-activity-%d", 5)),
				dag.If("cond2",
					dag.S(fmt.Sprintf("echo-activity-%d", 6)),
					dag.Block(dag.S(fmt.Sprintf("echo-activity-%d", 7)),
						dag.If("cond3",
							dag.S(fmt.Sprintf("echo-activity-%d", 8)),
							dag.S(fmt.Sprintf("echo-activity-%d", 8)),
						),
					),
				),
			),
		),
		dag.Block(
			dag.S(fmt.Sprintf("echo-activity-%d", 10)),
			dag.S(fmt.Sprintf("echo-activity-%d", 11)),
		),
		dag.S(fmt.Sprintf("echo-activity-%d", 11)),
	)

	err := dag.Build()
	require.NoError(t, err)
}
