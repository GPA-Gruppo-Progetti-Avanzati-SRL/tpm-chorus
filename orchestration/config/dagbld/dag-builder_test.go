package dagbld_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/dagbld"
	"testing"
)

func TestDagBuilder(t *testing.T) {
	DAGBuilder()
}

func DAGBuilder() dagbld.BlockStatement {
	var acts []dagbld.Statement
	acts = append(acts, dagbld.SimpleStatement{Nm: "start-activity"})
	for i := 1; i < 21; i++ {
		acts = append(acts, dagbld.SimpleStatement{Nm: fmt.Sprintf("echo-activity-%d", i)})
	}

	dag := dagbld.Block(
		acts[0],
		dagbld.Switch(
			dagbld.Case("case1", acts[1]),
			dagbld.Case("case2", acts[2]),
		),
		dagbld.IfStmt("condition",
			acts[3],
			dagbld.Block(
				acts[4],
				acts[5],
				dagbld.IfStmt("pippo",
					acts[6],
					dagbld.Block(acts[7], dagbld.IfStmt("caio",
						acts[8],
						acts[9]),
					),
				),
			),
		),
		dagbld.Block(acts[10], acts[11]),
		acts[12],
	)

	dagPaths := dag.Paths()
	for _, p := range dagPaths {
		fmt.Printf("[1] Path: %v\n", p)
	}

	dag1 := dagbld.BlockStatement{
		acts[0],
		&dagbld.SwitchStatement{
			{
				Cond: "case1",
				Stmt: acts[1],
			},
			{
				Cond: "case2",
				Stmt: acts[2],
			},
		},
		&dagbld.IfStatement{
			Cond: "condition",
			Then: acts[3],
			Else: dagbld.BlockStatement{
				acts[4],
				acts[5],
				dagbld.IfStatement{
					Cond: "pippo",
					Then: acts[6],
					Else: dagbld.BlockStatement{
						acts[7],
						dagbld.IfStatement{
							Cond: "caio",
							Then: acts[8],
							Else: acts[9],
						},
					},
				},
			},
		},
		dagbld.BlockStatement{acts[10], acts[11]},
		acts[12],
	}

	pths := dag1.Paths()
	for _, p := range pths {
		fmt.Printf("[2] Path: %v\n", p)
	}

	/*	current := dag1[0].Out()
		for i := 1; i < len(dag1); i++ {
			for _, out := range current {
				for _, in := range dag1[i].In() {
					p := config.Path{
						SourceName: out,
						TargetName: in,
						Constraint: "",
					}
					fmt.Printf("Path: %v\n", p)
				}
			}

			paths := dag1[i].Paths()
			for _, p := range paths {
				fmt.Printf("Path: %v\n", p)
			}
			current = dag1[i].Out()
		}
		fmt.Println(dag1)*/
	return dag1
}
