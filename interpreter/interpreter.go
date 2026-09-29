package interpreter

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"pop/ast"
	"pop/lexer"
	"pop/parser"
	"pop/ui"
)

type Value struct {
	Kind string
	IVal int64
	FVal float64
	SVal string
	BVal bool
	LVal []Value
}

func listVal(v []Value) Value {
	return Value{Kind: "list", LVal: v}
}

func (v Value) String() string {
	switch v.Kind {
	case "int":
		return fmt.Sprintf("%d", v.IVal)
	case "float":
		return fmt.Sprintf("%g", v.FVal)
	case "string":
		return v.SVal
	case "bool":
		if v.BVal {
			return "true"
		}
		return "false"
	case "list":
		result := "["
		for i, item := range v.LVal {
			if i > 0 {
				result += ", "
			}
			result += item.String()
		}
		result += "]"
		return result
	default:
		return "<nil>"
	}
}

func intVal(n int64) Value     { return Value{Kind: "int", IVal: n} }
func floatVal(f float64) Value { return Value{Kind: "float", FVal: f} }
func strVal(s string) Value    { return Value{Kind: "string", SVal: s} }
func boolVal(b bool) Value     { return Value{Kind: "bool", BVal: b} }

type Environment struct {
	vars   map[string]Value
	parent *Environment
}

func newEnv() *Environment {
	return &Environment{vars: make(map[string]Value)}
}

func newChildEnv(parent *Environment) *Environment {
	return &Environment{vars: make(map[string]Value), parent: parent}
}

func (e *Environment) set(name string, val Value) {
	e.vars[name] = val
}

func (e *Environment) setExisting(name string, val Value) bool {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = val
		return true
	}
	if e.parent != nil {
		return e.parent.setExisting(name, val)
	}
	return false
}

func (e *Environment) get(name string) (Value, bool) {
	if v, ok := e.vars[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.get(name)
	}
	return Value{}, false
}

type Interpreter struct {
	env             *Environment
	importedModules map[string]bool
	importedSymbols map[string]string
	functions       map[string]*ast.FuncDecl
	ui              *ui.UI
}

func New() *Interpreter {
	return &Interpreter{
		env:             newEnv(),
		importedModules: make(map[string]bool),
		importedSymbols: make(map[string]string),
		functions:       make(map[string]*ast.FuncDecl),
		ui:              ui.New(),
	}
}

func (interp *Interpreter) Run(prog *ast.Program) error {
	for _, stmt := range prog.Statements {
		if err := interp.execStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (interp *Interpreter) execReassign(n *ast.Reassign) error {
	_, exists := interp.env.get(n.Name)
	if !exists {
		return fmt.Errorf("line %d: undefined variable %q", n.Line, n.Name)
	}
	val, err := interp.evalExpr(n.Value)
	if err != nil {
		return err
	}
	interp.env.setExisting(n.Name, val)
	return nil
}

func (interp *Interpreter) execFuncDecl(n *ast.FuncDecl) error {
	interp.functions[n.Name] = n
	return nil
}

func (interp *Interpreter) execForNumStmt(n *ast.ForNumStmt) error {
	countVal, err := interp.evalExpr(n.Count)
	if err != nil {
		return err
	}
	if countVal.Kind != "int" {
		return fmt.Errorf("line %d: for count must be int", n.Line)
	}
	for i := int64(0); i < countVal.IVal; i++ {
		interp.env.set(n.Var, intVal(i))
		for _, stmt := range n.Body {
			if err := interp.execStatement(stmt); err != nil {
				if err == ErrBreak {
					delete(interp.env.vars, n.Var)
					return nil
				}
				return err
			}
		}
	}
	delete(interp.env.vars, n.Var)
	return nil
}

func (interp *Interpreter) execWhenStmt(n *ast.WhenStmt) error {
	const maxIter = 10_000_000
	iter := 0
	for {
		if iter > maxIter {
			return fmt.Errorf("line %d: infinite loop detected in 'when' (exceeded %d iterations)", n.Line, maxIter)
		}
		iter++
		cond, err := interp.evalExpr(n.Condition)
		if err != nil {
			return err
		}
		if cond.Kind != "bool" {
			return fmt.Errorf("line %d: when condition must be bool, got %s", n.Line, cond.Kind)
		}
		if !cond.BVal {
			break
		}
		for _, stmt := range n.Body {
			if err := interp.execStatement(stmt); err != nil {
				if err == ErrBreak {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

var ErrBreak = fmt.Errorf("break")

type ReturnError struct {
	Val Value
}

func (r *ReturnError) Error() string { return "return" }

func (interp *Interpreter) execForEachStmt(n *ast.ForEachStmt) error {
	iter, err := interp.evalExpr(n.Iterable)
	if err != nil {
		return err
	}
	if iter.Kind != "list" {
		return fmt.Errorf("line %d: for..with requires a list", n.Line)
	}
	for _, item := range iter.LVal {
		interp.env.set(n.Var, item)
		for _, stmt := range n.Body {
			if err := interp.execStatement(stmt); err != nil {
				if err == ErrBreak {
					return nil
				}
				return err
			}
		}
	}
	delete(interp.env.vars, n.Var)
	return nil
}

func (interp *Interpreter) execStatement(node ast.Node) error {
	switch n := node.(type) {
	case *ast.LibImport:
		return interp.execLibImport(n)
	case *ast.VarDecl:
		return interp.execVarDecl(n)
	case *ast.Reassign:
		return interp.execReassign(n)
	case *ast.WhenStmt:
		return interp.execWhenStmt(n)
	case *ast.IfStmt:
		return interp.execIfStmt(n)
	case *ast.PrintStmt:
		return interp.execPrintStmt(n)
	case *ast.FuncDecl:
		return interp.execFuncDecl(n)
	case *ast.ForNumStmt:
		return interp.execForNumStmt(n)
	case *ast.ForEachStmt:
		return interp.execForEachStmt(n)
	case *ast.BreakStmt:
		return ErrBreak
	case *ast.ReturnStmt:
		val, err := interp.evalExpr(n.Value)
		if err != nil {
			return err
		}
		return &ReturnError{Val: val}
	case *ast.AllocStmt:
		size, err := interp.evalExpr(n.Size)
		if err != nil {
			return err
		}
		if size.Kind != "int" {
			return fmt.Errorf("line %d: alloc size must be int", n.Line)
		}
		interp.env.set(n.Name, listVal(make([]Value, size.IVal)))
		return nil
	case *ast.FreeStmt:
		_, ok := interp.env.get(n.Name)
		if !ok {
			return fmt.Errorf("line %d: cannot free undefined variable %q", n.Line, n.Name)
		}
		delete(interp.env.vars, n.Name)
		return nil
	case *ast.IndexAssign:
		list, ok := interp.env.get(n.Name)
		if !ok {
			return fmt.Errorf("line %d: undefined variable %q", n.Line, n.Name)
		}
		if list.Kind != "list" {
			return fmt.Errorf("line %d: %q is not a list", n.Line, n.Name)
		}
		idx, err := interp.evalExpr(n.Index)
		if err != nil {
			return err
		}
		if idx.Kind != "int" {
			return fmt.Errorf("line %d: index must be int", n.Line)
		}
		i := idx.IVal
		if i < 0 || i >= int64(len(list.LVal)) {
			return fmt.Errorf("line %d: index %d out of range", n.Line, i)
		}
		val, err := interp.evalExpr(n.Value)
		if err != nil {
			return err
		}
		list.LVal[i] = val
		interp.env.set(n.Name, list)
		return nil
	case *ast.ExprStmt:
		_, err := interp.evalExpr(n.Expr)
		return err
	default:
		return fmt.Errorf("unknown statement type: %T", node)
	}
}

func (interp *Interpreter) execLibImport(n *ast.LibImport) error {
	interp.importedModules[n.Module] = true
	if n.Symbol != "" {
		interp.importedSymbols[n.Symbol] = n.Module
	}
	builtin := map[string]bool{"random": true, "ui": true, "file": true, "time": true}
	if builtin[n.Module] {
		return nil
	}
	paths := []string{
		n.Module + ".plib",
		"libs/" + n.Module + ".plib",
	}
	var src []byte
	var found bool
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			src = data
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("line %d: library %q not found — run: pop get %s", n.Line, n.Module, n.Module)
	}
	l := lexer.New(string(src))
	tokens, err := l.Tokenize()
	if err != nil {
		return fmt.Errorf("lib %s lexer error: %v", n.Module, err)
	}
	p := parser.New(tokens)
	prog, err := p.Parse()
	if err != nil {
		return fmt.Errorf("lib %s parse error: %v", n.Module, err)
	}
	for _, stmt := range prog.Statements {
		if fn, ok := stmt.(*ast.FuncDecl); ok {
			if fn.Exported {
				interp.functions[fn.Name] = fn
			}
		}
	}
	return nil
}

func (interp *Interpreter) execVarDecl(n *ast.VarDecl) error {
	val, err := interp.evalExpr(n.Value)
	if err != nil {
		return err
	}
	val, err = coerce(val, n.TypeName, n.Line)
	if err != nil {
		return err
	}
	interp.env.set(n.Name, val)
	return nil
}

func (interp *Interpreter) execIfStmt(n *ast.IfStmt) error {
	cond, err := interp.evalExpr(n.Condition)
	if err != nil {
		return err
	}
	if cond.Kind != "bool" {
		return fmt.Errorf("line %d: if condition must be bool, got %s", n.Line, cond.Kind)
	}
	if cond.BVal {
		for _, stmt := range n.Body {
			if err := interp.execStatement(stmt); err != nil {
				return err
			}
		}
	}
	return nil
}

func (interp *Interpreter) execPrintStmt(n *ast.PrintStmt) error {
	parts := make([]string, 0, len(n.Values))
	for _, expr := range n.Values {
		val, err := interp.evalExpr(expr)
		if err != nil {
			return err
		}
		parts = append(parts, val.String())
	}
	fmt.Println(strings.Join(parts, " "))
	return nil
}

func (interp *Interpreter) evalUnary(n *ast.UnaryExpr) (Value, error) {
	val, err := interp.evalExpr(n.Operand)
	if err != nil {
		return Value{}, err
	}
	switch n.Op {
	case "-":
		switch val.Kind {
		case "int":
			return intVal(-val.IVal), nil
		case "float":
			return floatVal(-val.FVal), nil
		default:
			return Value{}, fmt.Errorf("invalid unary - on %s", val.Kind)
		}
	case "!", "not":
		if val.Kind != "bool" {
			return Value{}, fmt.Errorf("invalid unary ! on %s", val.Kind)
		}
		return boolVal(!val.BVal), nil
	}
	return Value{}, fmt.Errorf("unknown unary operator %q", n.Op)
}

func (interp *Interpreter) evalExpr(node ast.Node) (Value, error) {
	switch n := node.(type) {
	case *ast.IntLit:
		return intVal(n.Value), nil
	case *ast.FloatLit:
		return floatVal(n.Value), nil
	case *ast.StringLit:
		return strVal(n.Value), nil
	case *ast.BoolLit:
		return boolVal(n.Value), nil
	case *ast.UnaryExpr:
		return interp.evalUnary(n)
	case *ast.IndexExpr:
		list, err := interp.evalExpr(n.List)
		if err != nil {
			return Value{}, err
		}
		idx, err := interp.evalExpr(n.Index)
		if err != nil {
			return Value{}, err
		}
		if list.Kind != "list" {
			return Value{}, fmt.Errorf("line %d: index on non-list", n.Line)
		}
		if idx.Kind != "int" {
			return Value{}, fmt.Errorf("line %d: index must be int", n.Line)
		}
		i := idx.IVal
		if i < 0 || i >= int64(len(list.LVal)) {
			return Value{}, fmt.Errorf("line %d: index %d out of range", n.Line, i)
		}
		return list.LVal[i], nil
	case *ast.ListLit:
		var vals []Value
		for _, elem := range n.Elements {
			v, err := interp.evalExpr(elem)
			if err != nil {
				return Value{}, err
			}
			vals = append(vals, v)
		}
		return listVal(vals), nil
	case *ast.Identifier:
		v, ok := interp.env.get(n.Name)
		if !ok {
			return Value{}, fmt.Errorf("line %d: undefined variable %q", n.Line, n.Name)
		}
		return v, nil
	case *ast.BinaryExpr:
		return interp.evalBinary(n)
	case *ast.CallExpr:
		return interp.evalCall(n)
	case *ast.RefExpr:
		v, ok := interp.env.get(n.Name)
		if !ok {
			return Value{}, fmt.Errorf("line %d: undefined variable %q", n.Line, n.Name)
		}
		return v, nil
	case *ast.DerefExpr:
		v, ok := interp.env.get(n.Name)
		if !ok {
			return Value{}, fmt.Errorf("line %d: undefined variable %q", n.Line, n.Name)
		}
		return v, nil
	default:
		return Value{}, fmt.Errorf("unknown expression type: %T", node)
	}
}

func (interp *Interpreter) evalBinary(n *ast.BinaryExpr) (Value, error) {
	left, err := interp.evalExpr(n.Left)
	if err != nil {
		return Value{}, err
	}
	right, err := interp.evalExpr(n.Right)
	if err != nil {
		return Value{}, err
	}

	if left.Kind == "float" || (right.Kind == "float" && left.Kind != "string" && right.Kind != "string") {
		lf := toFloat(left)
		rf := toFloat(right)
		switch n.Op {
		case "+":
			return floatVal(lf + rf), nil
		case "-":
			return floatVal(lf - rf), nil
		case "*":
			return floatVal(lf * rf), nil
		case "/":
			if rf == 0 {
				return Value{}, fmt.Errorf("division by zero")
			}
			return floatVal(lf / rf), nil
		case "<":
			return boolVal(lf < rf), nil
		case "<=":
			return boolVal(lf <= rf), nil
		case ">":
			return boolVal(lf > rf), nil
		case ">=":
			return boolVal(lf >= rf), nil
		case "==":
			return boolVal(lf == rf), nil
		case "!=":
			return boolVal(lf != rf), nil
		}
	}

	if left.Kind == "int" && right.Kind == "int" {
		switch n.Op {
		case "+":
			return intVal(left.IVal + right.IVal), nil
		case "-":
			return intVal(left.IVal - right.IVal), nil
		case "*":
			return intVal(left.IVal * right.IVal), nil
		case "/":
			if right.IVal == 0 {
				return Value{}, fmt.Errorf("division by zero")
			}
			return intVal(left.IVal / right.IVal), nil
		case "%":
			return intVal(left.IVal % right.IVal), nil
		case "<":
			return boolVal(left.IVal < right.IVal), nil
		case "<=":
			return boolVal(left.IVal <= right.IVal), nil
		case ">":
			return boolVal(left.IVal > right.IVal), nil
		case ">=":
			return boolVal(left.IVal >= right.IVal), nil
		case "==":
			return boolVal(left.IVal == right.IVal), nil
		case "!=":
			return boolVal(left.IVal != right.IVal), nil
		}
	}

	if left.Kind == "string" || right.Kind == "string" {
		switch n.Op {
		case "+":
			return strVal(left.String() + right.String()), nil
		case "==":
			return boolVal(left.String() == right.String()), nil
		case "!=":
			return boolVal(left.String() != right.String()), nil
		}
	}

	if left.Kind == "bool" && right.Kind == "bool" {
		switch n.Op {
		case "==":
			return boolVal(left.BVal == right.BVal), nil
		case "!=":
			return boolVal(left.BVal != right.BVal), nil
		case "and":
			return boolVal(left.BVal && right.BVal), nil
		case "or":
			return boolVal(left.BVal || right.BVal), nil
		}
	}

	return Value{}, fmt.Errorf("unsupported operation %q on %s and %s", n.Op, left.Kind, right.Kind)
}

func (interp *Interpreter) evalCall(n *ast.CallExpr) (Value, error) {
	module := n.Module
	funcName := n.Func
	if module == "" {
		if mod, ok := interp.importedSymbols[funcName]; ok {
			module = mod
		}
	}
	switch module {
	case "random":
		return interp.callRandom(funcName, n.Args, n.Line)
	case "ui":
		return interp.callUI(funcName, n.Args, n.Line)
	case "file":
		return interp.callFile(funcName, n.Args, n.Line)
	case "time":
		return interp.callTime(funcName, n.Args, n.Line)
	case "":
		if fn, ok := interp.functions[funcName]; ok {
			return interp.callUserFunc(fn, n.Args, n.Line)
		}
		return interp.callBuiltin(funcName, n.Args, n.Line)
	default:
		return Value{}, fmt.Errorf("line %d: unknown module %q", n.Line, module)
	}
}

func (interp *Interpreter) callUI(fn string, args []ast.CallArg, line int) (Value, error) {
	named := make(map[string]Value)
	for _, arg := range args {
		val, err := interp.evalExpr(arg.Value)
		if err != nil {
			return Value{}, err
		}
		if arg.Name != "" {
			named[arg.Name] = val
		}
	}
	getStr := func(key string) string {
		if v, ok := named[key]; ok {
			return v.String()
		}
		return ""
	}
	getFloat := func(key string, def float32) float32 {
		if v, ok := named[key]; ok {
			if v.Kind == "int" {
				return float32(v.IVal)
			}
			if v.Kind == "float" {
				return float32(v.FVal)
			}
		}
		return def
	}

	switch fn {
	case "window":
		interp.ui.Window(getStr("title"), getFloat("width", 400), getFloat("height", 300))
		return Value{}, nil
	case "label":
		text := getStr("text")
		name := getStr("name")
		if name != "" {
			interp.ui.LabelNamed(name, text)
		} else {
			interp.ui.Label(text)
		}
		return Value{}, nil
	case "button":
		text := getStr("text")
		name := getStr("name")
		actionName := getStr("action")
		cb := func(a []string) {
			if fn, ok := interp.functions[actionName]; ok {
				interp.callUserFunc(fn, nil, line)
				interp.ui.Refresh()
			}
		}
		if name != "" {
			interp.ui.ButtonNamed(name, text, cb)
		} else {
			interp.ui.Button(text, cb)
		}
		return Value{}, nil
	case "checkbox":
		text := getStr("text")
		actionName := getStr("action")
		interp.ui.Checkbox(text, func(a []string) {
			if fn, ok := interp.functions[actionName]; ok {
				argVal := []ast.CallArg{{Value: &ast.StringLit{Value: a[0]}}}
				interp.callUserFunc(fn, argVal, line)
				interp.ui.Refresh()
			}
		})
		return Value{}, nil
	case "input":
		name := getStr("name")
		placeholder := getStr("placeholder")
		onChangeName := getStr("on_change")
		interp.ui.Input(name, placeholder, func(a []string) {
			if onChangeName == "" {
				return
			}
			if fn, ok := interp.functions[onChangeName]; ok {
				argVal := []ast.CallArg{{Value: &ast.StringLit{Value: a[0]}}}
				interp.callUserFunc(fn, argVal, line)
				interp.ui.Refresh()
			}
		})
		return Value{}, nil
	case "get":
		return strVal(interp.ui.Get(getStr("name"))), nil
	case "set":
		interp.ui.Set(getStr("name"), getStr("value"))
		interp.ui.Refresh()
		return Value{}, nil
	case "space":
		interp.ui.Space()
		return Value{}, nil
	case "run":
		interp.ui.Run()
		return Value{}, nil
	default:
		return Value{}, fmt.Errorf("line %d: unknown ui function %q", line, fn)
	}
}

func (interp *Interpreter) callFile(fn string, args []ast.CallArg, line int) (Value, error) {
	named := make(map[string]Value)
	for _, arg := range args {
		val, err := interp.evalExpr(arg.Value)
		if err != nil {
			return Value{}, err
		}
		if arg.Name != "" {
			named[arg.Name] = val
		}
	}
	getStr := func(key string) string {
		if v, ok := named[key]; ok {
			return v.String()
		}
		return ""
	}
	switch fn {
	case "read":
		data, err := os.ReadFile(getStr("path"))
		if err != nil {
			return strVal(""), nil
		}
		return strVal(string(data)), nil
	case "write":
		err := os.WriteFile(getStr("path"), []byte(getStr("content")), 0644)
		return boolVal(err == nil), nil
	case "append":
		f, err := os.OpenFile(getStr("path"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return boolVal(false), nil
		}
		defer f.Close()
		f.WriteString(getStr("content"))
		return boolVal(true), nil
	case "delete":
		err := os.Remove(getStr("path"))
		return boolVal(err == nil), nil
	case "exists":
		_, err := os.Stat(getStr("path"))
		return boolVal(!os.IsNotExist(err)), nil
	case "print":
		data, err := os.ReadFile(getStr("path"))
		if err != nil {
			return boolVal(false), nil
		}
		fmt.Println(string(data))
		return boolVal(true), nil
	default:
		return Value{}, fmt.Errorf("line %d: unknown file function %q", line, fn)
	}
}

func (interp *Interpreter) callTime(fn string, args []ast.CallArg, line int) (Value, error) {
	switch fn {
	case "now":
		return strVal(time.Now().Format("2006-01-02 15:04:05")), nil
	case "date":
		return strVal(time.Now().Format("2006-01-02")), nil
	case "clock":
		return strVal(time.Now().Format("15:04:05")), nil
	case "unix":
		return intVal(time.Now().Unix()), nil
	case "sleep":
		named := make(map[string]Value)
		for _, arg := range args {
			val, err := interp.evalExpr(arg.Value)
			if err != nil {
				return Value{}, err
			}
			if arg.Name != "" {
				named[arg.Name] = val
			}
		}
		if v, ok := named["ms"]; ok && v.Kind == "int" {
			time.Sleep(time.Duration(v.IVal) * time.Millisecond)
		}
		if v, ok := named["s"]; ok && v.Kind == "int" {
			time.Sleep(time.Duration(v.IVal) * time.Second)
		}
		return Value{}, nil
	default:
		return Value{}, fmt.Errorf("line %d: unknown time function %q", line, fn)
	}
}

func (interp *Interpreter) callUserFunc(fn *ast.FuncDecl, args []ast.CallArg, line int) (Value, error) {
	if args == nil {
		args = []ast.CallArg{}
	}
	if len(args) != len(fn.Params) {
		return Value{}, fmt.Errorf("line %d: function %s expects %d args", line, fn.Name, len(fn.Params))
	}
	local := newChildEnv(interp.env)
	for i, param := range fn.Params {
		val, err := interp.evalExpr(args[i].Value)
		if err != nil {
			return Value{}, err
		}
		local.set(param, val)
	}
	oldEnv := interp.env
	interp.env = local
	defer func() { interp.env = oldEnv }()
	for _, stmt := range fn.Body {
		if err := interp.execStatement(stmt); err != nil {
			if ret, ok := err.(*ReturnError); ok {
				interp.env = oldEnv
				return ret.Val, nil
			}
			return Value{}, err
		}
	}
	return Value{}, nil
}

func (interp *Interpreter) callRandom(fn string, args []ast.CallArg, line int) (Value, error) {
	switch fn {
	case "number":
		from, to, err := interp.resolveFromTo(args, line)
		if err != nil {
			return Value{}, err
		}
		return intVal(from + rand.Int63n(to-from+1)), nil
	default:
		return Value{}, fmt.Errorf("line %d: unknown function random.%s", line, fn)
	}
}

func (interp *Interpreter) callBuiltin(fn string, args []ast.CallArg, line int) (Value, error) {
	switch fn {
	case "inputw":
		if len(args) == 1 {
			prompt, err := interp.evalExpr(args[0].Value)
			if err != nil {
				return Value{}, err
			}
			fmt.Print(prompt.String())
		}
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimRight(input, "\r\n")
		return strVal(input), nil

	case "int":
		if len(args) != 1 {
			return Value{}, fmt.Errorf("line %d: int() requires 1 argument", line)
		}
		val, err := interp.evalExpr(args[0].Value)
		if err != nil {
			return Value{}, err
		}
		switch val.Kind {
		case "int":
			return val, nil
		case "float":
			return intVal(int64(val.FVal)), nil
		case "string":
			n, err := strconv.ParseInt(val.SVal, 10, 64)
			if err != nil {
				return Value{}, fmt.Errorf("line %d: cannot convert %q to int", line, val.SVal)
			}
			return intVal(n), nil
		case "bool":
			if val.BVal {
				return intVal(1), nil
			}
			return intVal(0), nil
		}
		return Value{}, fmt.Errorf("line %d: cannot convert %s to int", line, val.Kind)

	case "float":
		if len(args) != 1 {
			return Value{}, fmt.Errorf("line %d: float() requires 1 argument", line)
		}
		val, err := interp.evalExpr(args[0].Value)
		if err != nil {
			return Value{}, err
		}
		switch val.Kind {
		case "float":
			return val, nil
		case "int":
			return floatVal(float64(val.IVal)), nil
		case "string":
			f, err := strconv.ParseFloat(val.SVal, 64)
			if err != nil {
				return Value{}, fmt.Errorf("line %d: cannot convert %q to float", line, val.SVal)
			}
			return floatVal(f), nil
		}
		return Value{}, fmt.Errorf("line %d: cannot convert %s to float", line, val.Kind)

	case "string":
		if len(args) != 1 {
			return Value{}, fmt.Errorf("line %d: string() requires 1 argument", line)
		}
		val, err := interp.evalExpr(args[0].Value)
		if err != nil {
			return Value{}, err
		}
		return strVal(val.String()), nil

	case "bool":
		if len(args) != 1 {
			return Value{}, fmt.Errorf("line %d: bool() requires 1 argument", line)
		}
		val, err := interp.evalExpr(args[0].Value)
		if err != nil {
			return Value{}, err
		}
		switch val.Kind {
		case "bool":
			return val, nil
		case "int":
			return boolVal(val.IVal != 0), nil
		case "string":
			return boolVal(val.SVal != ""), nil
		}
		return Value{}, fmt.Errorf("line %d: cannot convert %s to bool", line, val.Kind)
	}

	return Value{}, fmt.Errorf("line %d: unknown function %q", line, fn)
}

func (interp *Interpreter) resolveFromTo(args []ast.CallArg, line int) (int64, int64, error) {
	named := make(map[string]Value)
	for _, arg := range args {
		val, err := interp.evalExpr(arg.Value)
		if err != nil {
			return 0, 0, err
		}
		if arg.Name != "" {
			named[arg.Name] = val
		}
	}
	fromV, ok1 := named["from"]
	toV, ok2 := named["to"]
	if !ok1 || !ok2 {
		return 0, 0, fmt.Errorf("line %d: random.number requires named args 'from' and 'to'", line)
	}
	from := toInt(fromV)
	to := toInt(toV)
	if from > to {
		return 0, 0, fmt.Errorf("line %d: 'from' must be <= 'to'", line)
	}
	return from, to, nil
}

func toFloat(v Value) float64 {
	switch v.Kind {
	case "int":
		return float64(v.IVal)
	case "float":
		return v.FVal
	}
	return 0
}

func toInt(v Value) int64 {
	switch v.Kind {
	case "int":
		return v.IVal
	case "float":
		return int64(v.FVal)
	}
	return 0
}

func coerce(v Value, typeName string, line int) (Value, error) {
	switch typeName {
	case "int":
		switch v.Kind {
		case "int":
			return v, nil
		case "float":
			return intVal(int64(v.FVal)), nil
		}
	case "float":
		switch v.Kind {
		case "float":
			return v, nil
		case "int":
			return floatVal(float64(v.IVal)), nil
		}
	case "string":
		if v.Kind == "string" {
			return v, nil
		}
	case "bool":
		if v.Kind == "bool" {
			return v, nil
		}
	case "list":
		if v.Kind == "list" {
			return v, nil
		}
	}
	return Value{}, fmt.Errorf("line %d: cannot assign %s to %s", line, v.Kind, typeName)
}
