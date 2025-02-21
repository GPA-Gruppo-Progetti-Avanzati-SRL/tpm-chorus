package dagbld

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
)

type IfStatement struct {
	Cond    string
	Ingress Statement
	Then    Statement
	Else    Statement
	Egress  Statement
}

func (stmt IfStatement) Name() string {
	return stmt.Cond
}

func (stmt IfStatement) Type() string {
	return StatementTypeIf
}

func (stmt IfStatement) In() InputOutput {
	return stmt.Ingress.In()
}

func (stmt IfStatement) Out() InputOutput {
	return stmt.Egress.Out()
}

func (stmt IfStatement) Paths() []config.Path {

	var paths []config.Path

	current := stmt.Ingress.Out()

	// was 	if stmt.Then.Type() != StatementTypeGoto
	if stmt.Then.In().Type() != StatementTypeGoto {
		p := config.Path{
			SourceName: current.Name,
			TargetName: stmt.Then.In().Name,
			Constraint: stmt.Cond,
		}
		paths = append(paths, p)

		if stmt.Then.Out().Type() != StatementTypeGoto {
			p = config.Path{
				SourceName: stmt.Then.Out().Name,
				TargetName: stmt.Egress.In().Name,
				Constraint: "",
			}
			paths = append(paths, p)
		}

		thenPath := stmt.Then.Paths()
		if thenPath != nil {
			paths = append(paths, thenPath...)
		}
	} else {
		p := config.Path{
			SourceName: current.Name,
			TargetName: stmt.Then.Out().Name,
			Constraint: stmt.Cond,
		}
		paths = append(paths, p)
	}

	if stmt.Else != nil {
		if stmt.Else.Type() != StatementTypeGoto {
			p := config.Path{
				SourceName: current.Name,
				TargetName: stmt.Else.In().Name,
				Constraint: "",
			}
			paths = append(paths, p)

			if stmt.Else.Out().Type() != StatementTypeGoto {
				p = config.Path{
					SourceName: stmt.Else.Out().Name,
					TargetName: stmt.Egress.In().Name,
					Constraint: "",
				}
				paths = append(paths, p)
			}

			elsePath := stmt.Else.Paths()
			if elsePath != nil {
				paths = append(paths, elsePath...)
			}
		} else {
			p := config.Path{
				SourceName: current.Name,
				TargetName: stmt.Else.Out().Name,
				Constraint: "",
			}
			paths = append(paths, p)
		}
	}

	return paths
}
