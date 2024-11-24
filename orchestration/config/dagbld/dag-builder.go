package dagbld

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"strings"
)

type InputOutput struct {
	Name     string
	Cond     string
	StmtType string
}

func (i InputOutput) IsNilish() bool {
	return i.Name == ""
}

func (i InputOutput) Type() string {
	return i.StmtType
}

const (
	StatementTypeSimple = "simple"
	StatementTypeGoto   = "goto"
	StatementTypeIf     = "if"
	StatementTypeSwitch = "switch"
	StatementTypeCase   = "case"
	StatementTypeBlock  = "block"
)

type Statement interface {
	Name() string
	In() InputOutput
	Out() InputOutput
	Paths() []config.Path
	Type() string
}

type SimpleStatement struct {
	Nm string
}

func (stmt SimpleStatement) Name() string {
	return stmt.Nm
}

func (stmt SimpleStatement) Type() string {
	return StatementTypeSimple
}

func (stmt SimpleStatement) In() InputOutput {
	return InputOutput{stmt.Nm, "", StatementTypeSimple}
}

func (stmt SimpleStatement) Out() InputOutput {
	return InputOutput{stmt.Nm, "", StatementTypeSimple}
}

func (stmt SimpleStatement) Paths() []config.Path {
	return nil
}

type GotoStatement struct {
	Nm string
}

func (stmt GotoStatement) Name() string {
	return stmt.Nm
}

func (stmt GotoStatement) Type() string {
	return StatementTypeGoto
}

func (stmt GotoStatement) In() InputOutput {
	return InputOutput{StmtType: StatementTypeGoto}
}

func (stmt GotoStatement) Out() InputOutput {
	return InputOutput{stmt.Nm, "", StatementTypeGoto}
}

func (stmt GotoStatement) Paths() []config.Path {
	return nil
}

/*func S(a Activity) Statement {
	return SimpleStatement{Nm: a.Name()}
}*/

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

type CaseStatement struct {
	Cond string
	Stmt Statement
}

func (c CaseStatement) Name() string {
	return c.Cond
}

func (stmt CaseStatement) Type() string {
	return StatementTypeCase
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
	for _, c := range stmt.Cases {
		p := config.Path{
			SourceName: stmt.Ingress.Out().Name,
			TargetName: c.In().Name,
			Constraint: c.Cond,
		}
		paths = append(paths, p)

		p = config.Path{
			SourceName: c.Out().Name,
			TargetName: stmt.Egress.Name(),
			Constraint: "",
		}
		paths = append(paths, p)

		paths = append(paths, c.Paths()...)
	}

	return paths
}

/*func (dag B) Add(stmts ...Statement) B {
	for _, s := range stmts {
		dag = append(dag, s)
	}

	return dag
}*/

/*func S(n string) Statement {
	return &SimpleStatement{Nm: n}
}

func If(cond string, thenStmt Statement, elseStmt Statement) Statement {
	return &IfStatement{Cond: cond, Then: thenStmt, Else: elseStmt}
}

func Block(stmts ...Statement) BlockStatement {
	cs := BlockStatement{}
	for _, s := range stmts {
		cs = append(cs, s)
	}

	return cs
}

func Case(cond string, stmt Statement) CaseStatement {
	return CaseStatement{
		Cond: cond,
		Stmt: stmt,
	}
}

func Switch(cas ...CaseStatement) Statement {
	stmt := SwitchStatement{}
	stmt.Cases = append(stmt.Cases, cas...)
	return &stmt
}*/

type DAGBuilder struct {
	f    DagModel
	stmt BlockStatement
}

func (dag *DAGBuilder) With(s ...Statement) {
	dag.stmt = append(dag.stmt, s...)
}

func (dag *DAGBuilder) S(n string) Statement {
	return &SimpleStatement{Nm: n}
}

func (dag *DAGBuilder) Goto(n string) Statement {
	return &GotoStatement{Nm: n}
}

func (dag *DAGBuilder) Switch(cas ...CaseStatement) Statement {
	stmt := SwitchStatement{
		Ingress: SimpleStatement{Nm: dag.f.AddNopActivity("Switch")},
		Egress:  SimpleStatement{Nm: dag.f.AddNopActivity("End")},
	}
	stmt.Cases = append(stmt.Cases, cas...)
	return &stmt
}

func (dag *DAGBuilder) Case(cond string, stmt Statement) CaseStatement {
	return CaseStatement{
		Cond: cond,
		Stmt: stmt,
	}
}

func (dag *DAGBuilder) Nop(description string) Statement {
	return SimpleStatement{Nm: dag.f.AddNopActivity(description)}
}

func (dag *DAGBuilder) Block(stmts ...Statement) BlockStatement {
	cs := BlockStatement{}
	for _, s := range stmts {
		cs = append(cs, s)
	}

	return cs
}

func (dag *DAGBuilder) If(cond string, thenStmt Statement, elseStmt Statement) Statement {
	stmt := IfStatement{
		Ingress: SimpleStatement{Nm: dag.f.AddNopActivity("If")},
		Egress:  SimpleStatement{Nm: dag.f.AddNopActivity("End")},
		Cond:    cond,
		Then:    thenStmt,
		Else:    elseStmt,
	}

	return &stmt
}

func (dag *DAGBuilder) Build(optimize bool) error {

	dagPaths := dag.stmt.Paths()
	dagPaths = removeDups(dagPaths)

	for _, p := range dagPaths {
		err := dag.f.AddPath(p.SourceName, p.TargetName, p.Constraint)
		if err != nil {
			return err
		}
	}

	if optimize {
		return dag.f.Optimize()
	}
	return nil
}

func removeDups(paths []config.Path) []config.Path {
	m := make(map[string]struct{})
	var uniquePaths []config.Path
	for _, p := range paths {
		n := strings.Join([]string{p.SourceName, p.TargetName, p.Constraint}, "#")
		if _, ok := m[n]; !ok {
			m[n] = struct{}{}
			uniquePaths = append(uniquePaths, p)
		}
	}

	return uniquePaths
}

type DagModel interface {
	Optimize() error
	AddNopActivity(d string) string
	AddPath(src, target, condition string) error
}

func NewDAGPathBuilder(f DagModel) *DAGBuilder {
	return &DAGBuilder{f: f}
}

//func DAGBuilder() B {
//	var acts []Statement
//	acts = append(acts, SimpleStatement{Nm: "start-activity"})
//	for i := 0; i < 10; i++ {
//		acts = append(acts, SimpleStatement{Nm: fmt.Sprintf("echo-activity-%d", i)})
//	}
//
//	dag := Block(
//		acts[0],
//		IfStmt("condition",
//			acts[1],
//			Block(
//				acts[2],
//				acts[3],
//				IfStmt("pippo",
//					acts[4],
//					Block(acts[5], IfStmt("caio",
//						acts[6],
//						acts[7]),
//					),
//				),
//			),
//		),
//		Block(acts[8], acts[9]),
//		acts[10],
//	)
//
//	dag1 := B{
//		acts[0],
//		&If{
//			Cond: "condition",
//			Then: acts[1],
//			Else: B{
//				acts[2],
//				acts[3],
//				If{
//					Cond: "pippo",
//					Then: acts[4],
//					Else: B{
//						acts[5],
//						If{
//							Cond: "caio",
//							Then: acts[6],
//							Else: acts[7],
//						},
//					},
//				},
//			},
//		},
//		B{acts[8], acts[9]},
//		acts[10],
//	}
//
//	start := dag1[0]
//	for i := 1; i < len(dag1); i++ {
//		fmt.Printf("From: %v, %v\n", start, dag[i].In())
//	}
//
//	fmt.Println(dag1)
//	return dag
//}
