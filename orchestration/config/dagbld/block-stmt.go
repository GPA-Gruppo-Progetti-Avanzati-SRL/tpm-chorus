package dagbld

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
)

type BlockStatement []Statement

func (stmt BlockStatement) Name() string {
	return stmt[0].Name()
}

func (stmt BlockStatement) Type() string {
	return StatementTypeBlock
}

func (stmt BlockStatement) In() InputOutput {
	v := stmt[0].In()
	return v
}

func (stmt BlockStatement) Out() InputOutput {
	v := stmt[len(stmt)-1].Out()
	return v
}

func (stmt BlockStatement) Paths() []config.Path {
	var paths []config.Path
	paths = append(paths, stmt[0].Paths()...)
	current := stmt[0].Out()
	for i := 1; i < len(stmt); i++ {
		if stmt[i].Type() != StatementTypeGoto {
			p := config.Path{
				SourceName: current.Name,
				TargetName: stmt[i].In().Name,
				Constraint: stmt[i].In().Cond,
			}
			paths = append(paths, p)
			paths = append(paths, stmt[i].Paths()...)
			current = stmt[i].Out()
		} else {
			p := config.Path{
				SourceName: current.Name,
				TargetName: stmt[i].Out().Name,
				Constraint: "",
			}
			paths = append(paths, p)
			break
		}
	}
	return paths
}
