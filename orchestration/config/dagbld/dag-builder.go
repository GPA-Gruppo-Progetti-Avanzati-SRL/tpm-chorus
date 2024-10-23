package dagbld

import "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"

type Input struct {
	Name string
	Cond string
}

type Statement interface {
	Name() string
	In() []Input
	Out() []string
	Paths() []config.Path
}

type SimpleStatement struct {
	Nm string
}

func (stmt SimpleStatement) Name() string {
	return stmt.Nm
}

func (stmt SimpleStatement) In() []Input {
	return []Input{{stmt.Nm, ""}}
}

func (stmt SimpleStatement) Out() []string {
	return []string{stmt.Nm}
}

func (stmt SimpleStatement) Paths() []config.Path {
	return nil
}

/*func S(a Activity) Statement {
	return SimpleStatement{Nm: a.Name()}
}*/

type BlockStatement []Statement

func (stmt BlockStatement) Name() string {
	return stmt[0].Name()
}

func (stmt BlockStatement) In() []Input {
	return stmt[0].In()
}

func (stmt BlockStatement) Out() []string {
	return stmt[len(stmt)-1].Out()
}

func (stmt BlockStatement) Paths() []config.Path {

	var paths []config.Path
	current := stmt[0].Out()
	for i := 1; i < len(stmt); i++ {
		for _, out := range current {
			for _, in := range stmt[i].In() {
				p := config.Path{
					SourceName: out,
					TargetName: in.Name,
					Constraint: in.Cond,
				}
				paths = append(paths, p)
			}
		}
		paths = append(paths, stmt[i].Paths()...)
		current = stmt[i].Out()
	}
	return paths
}

type IfStatement struct {
	Cond string
	Then Statement
	Else Statement
}

func (stmt IfStatement) Name() string {
	return stmt.Cond
}

func (stmt IfStatement) In() []Input {

	var in []Input
	for _, i := range stmt.Then.In() {
		in = append(in, Input{i.Name, stmt.Cond})
	}

	if stmt.Else != nil {
		in = append(in, stmt.Else.In()...)
	}
	return in
}

func (stmt IfStatement) Out() []string {
	out := stmt.Then.Out()
	if stmt.Else != nil {
		out = append(out, stmt.Else.Out()...)
	}

	return out
}

func (stmt IfStatement) Paths() []config.Path {

	var paths []config.Path

	thenPath := stmt.Then.Paths()
	if thenPath != nil {
		paths = append(paths, thenPath...)
	}
	if stmt.Else != nil {
		elsePath := stmt.Else.Paths()
		if elsePath != nil {
			paths = append(paths, elsePath...)
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

func (c CaseStatement) In() []Input {

	var in []Input
	for _, i := range c.Stmt.In() {
		in = append(in, Input{i.Name, c.Cond})
	}
	return in
}

func (c CaseStatement) Out() []string {
	return c.Stmt.Out()
}

func (c CaseStatement) Paths() []config.Path {
	return c.Stmt.Paths()
}

type SwitchStatement []CaseStatement

func (stmt SwitchStatement) Name() string {
	return ""
}

func (stmt SwitchStatement) In() []Input {
	var in []Input

	for _, c := range stmt {
		in = append(in, c.In()...)
	}

	return in
}

func (stmt SwitchStatement) Out() []string {
	var out []string

	for _, c := range stmt {
		out = append(out, c.Out()...)
	}

	return out
}

func (stmt SwitchStatement) Paths() []config.Path {

	var paths []config.Path

	for _, c := range stmt {
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
	stmt = append(stmt, cas...)
	return &stmt
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
