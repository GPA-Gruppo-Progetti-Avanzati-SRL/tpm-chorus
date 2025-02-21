package dagbld

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
)

type CaseStatement struct {
	Cond string
	Stmt Statement
}

func (c CaseStatement) Name() string {
	return c.Cond
}

func (c CaseStatement) Type() string {
	return c.Stmt.Type() // Was StatementTypeCase
}

func (c CaseStatement) In() InputOutput {
	return c.Stmt.In()
}

func (c CaseStatement) Out() InputOutput {
	v := c.Stmt.Out()
	return v
}

func (c CaseStatement) Paths() []config.Path {
	return c.Stmt.Paths()
}

type SwitchStatement struct {
	Ingress Statement
	Cases   []CaseStatement
	Egress  Statement
}

func (stmt SwitchStatement) Name() string {
	return ""
}

func (stmt SwitchStatement) Type() string {
	return StatementTypeSwitch
}

func (stmt SwitchStatement) In() InputOutput {
	return stmt.Ingress.In()
}

func (stmt SwitchStatement) Out() InputOutput {
	return stmt.Egress.Out()
}

func (stmt SwitchStatement) Paths() []config.Path {

	var paths []config.Path

	current := stmt.Ingress.Out()
	for _, c := range stmt.Cases {

		if c.Type() != StatementTypeGoto {
			p := config.Path{
				SourceName: current.Name,
				TargetName: c.In().Name,
				Constraint: c.Cond,
			}
			paths = append(paths, p)

			if c.Out().Type() != StatementTypeGoto {
				p = config.Path{
					SourceName: c.Out().Name,
					TargetName: stmt.Egress.Name(),
					Constraint: "",
				}
				paths = append(paths, p)
			}

			paths = append(paths, c.Paths()...)
		} else {
			p := config.Path{
				SourceName: current.Name,
				TargetName: c.Out().Name,
				Constraint: c.Cond,
			}
			paths = append(paths, p)
		}

		//p := config.Path{
		//	SourceName: stmt.Ingress.Out().Name,
		//	TargetName: c.In().Name,
		//	Constraint: c.Cond,
		//}
		//paths = append(paths, p)

		//p = config.Path{
		//	SourceName: c.Out().Name,
		//	TargetName: stmt.Egress.Name(),
		//	Constraint: "",
		//}
		//paths = append(paths, p)

	}

	return paths
}
