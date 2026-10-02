package ast

import "fmt"

type Node interface {
	nodeType() string
}

type Program struct {
	Statements []Node
	ARMemory   bool
}

func (p *Program) nodeType() string { return "Program" }

type VarDecl struct {
	TypeName string
	Name     string
	Value    Node
	Line     int
}

func (v *VarDecl) nodeType() string { return "VarDecl" }

type Reassign struct {
	Name  string
	Value Node
	Line  int
}

func (r *Reassign) nodeType() string { return "Reassign" }

type IndexAssign struct {
	Name  string
	Index Node
	Value Node
	Line  int
}

func (i *IndexAssign) nodeType() string { return "IndexAssign" }

type LibImport struct {
	Module string
	Symbol string
	Line   int
}

func (l *LibImport) nodeType() string { return "LibImport" }

type IfStmt struct {
	Condition Node
	Body      []Node
	ElseIf    []*ElseIfClause
	Else      []Node
	Line      int
}

func (i *IfStmt) nodeType() string { return "IfStmt" }

type ElseIfClause struct {
	Condition Node
	Body      []Node
}

func (e *ElseIfClause) nodeType() string { return "ElseIfClause" }

type WhenStmt struct {
	Condition Node
	Body      []Node
	Line      int
}

func (w *WhenStmt) nodeType() string { return "WhenStmt" }

type ForEachStmt struct {
	Var      string
	Iterable Node
	Body     []Node
	Line     int
}

func (f *ForEachStmt) nodeType() string { return "ForEachStmt" }

type ForNumStmt struct {
	Var   string
	Count Node
	Body  []Node
	Line  int
}

func (f *ForNumStmt) nodeType() string { return "ForNumStmt" }

type PrintStmt struct {
	Values []Node
	Line   int
}

func (p *PrintStmt) nodeType() string { return "PrintStmt" }

type ReturnStmt struct {
	Value Node
	Line  int
}

func (r *ReturnStmt) nodeType() string { return "ReturnStmt" }

type BreakStmt struct {
	Line int
}

func (b *BreakStmt) nodeType() string { return "BreakStmt" }

type FuncDecl struct {
	Name     string
	Params   []string
	Body     []Node
	Exported bool
	Line     int
}

func (f *FuncDecl) nodeType() string { return "FuncDecl" }

type FreeStmt struct {
	Name string
	Line int
}

func (f *FreeStmt) nodeType() string { return "FreeStmt" }

type AllocStmt struct {
	Name string
	Size Node
	Line int
}

func (a *AllocStmt) nodeType() string { return "AllocStmt" }

type ExprStmt struct {
	Expr Node
}

func (e *ExprStmt) nodeType() string { return "ExprStmt" }

type IntLit struct {
	Value int64
}

func (i *IntLit) nodeType() string { return "IntLit" }

type FloatLit struct {
	Value float64
}

func (f *FloatLit) nodeType() string { return "FloatLit" }

type StringLit struct {
	Value string
}

func (s *StringLit) nodeType() string { return "StringLit" }

type BoolLit struct {
	Value bool
}

func (b *BoolLit) nodeType() string { return "BoolLit" }

type ListLit struct {
	Elements []Node
}

func (l *ListLit) nodeType() string { return "ListLit" }

type Identifier struct {
	Name string
	Line int
}

func (i *Identifier) nodeType() string { return "Identifier" }

type IndexExpr struct {
	List  Node
	Index Node
	Line  int
}

func (i *IndexExpr) nodeType() string { return "IndexExpr" }

type BinaryExpr struct {
	Op    string
	Left  Node
	Right Node
}

func (b *BinaryExpr) nodeType() string { return "BinaryExpr" }

type UnaryExpr struct {
	Op      string
	Operand Node
}

func (u *UnaryExpr) nodeType() string { return "UnaryExpr" }

type CallExpr struct {
	Module string
	Func   string
	Args   []CallArg
	Line   int
}

func (c *CallExpr) nodeType() string { return "CallExpr" }

func (c *CallExpr) String() string {
	if c.Module != "" {
		return fmt.Sprintf("%s.%s(...)", c.Module, c.Func)
	}
	return fmt.Sprintf("%s(...)", c.Func)
}

type CallArg struct {
	Name  string
	Value Node
}

type RefExpr struct {
	Name string
	Line int
}

func (r *RefExpr) nodeType() string { return "RefExpr" }

type DerefExpr struct {
	Name string
	Line int
}

func (d *DerefExpr) nodeType() string { return "DerefExpr" }

type MapLit struct {
    Keys   []Node
    Values []Node
    Line   int
}

type MapAccessExpr struct {
    Name  string
    Key   Node
    Line  int
}

type MapAssign struct {
    Name  string
    Key   Node
    Value Node
    Line  int
}
func (m *MapLit) nodeType() string { return "MapLit" }
func (m *MapAccessExpr) nodeType() string { return "MapAccessExpr" }
func (m *MapAssign) nodeType() string { return "MapAssign" }
