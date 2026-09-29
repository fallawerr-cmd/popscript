package parser

import (
	"fmt"
	"strconv"
	"strings"

	"pop/ast"
	"pop/lexer"
)

type Parser struct {
	tokens []lexer.Token
	pos    int
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) peek() lexer.Token {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == lexer.TOKEN_NEWLINE {
		p.pos++
	}
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() lexer.Token {
	t := p.peek()
	p.pos++
	return t
}

func (p *Parser) expect(tt lexer.TokenType) (lexer.Token, error) {
	t := p.peek()
	if t.Type != tt {
		return t, fmt.Errorf("line %d: expected %s, got %s (%q)", t.Line, tt, t.Type, t.Value)
	}
	p.pos++
	return t, nil
}

func (p *Parser) skipNewlines() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == lexer.TOKEN_NEWLINE {
		p.pos++
	}
}

func (p *Parser) Parse() (*ast.Program, error) {
	prog := &ast.Program{}
	p.skipNewlines()
	if p.peek().Type == lexer.TOKEN_ARMEMORY {
		p.advance()
		if _, err := p.expect(lexer.TOKEN_ASSIGN); err != nil {
			return nil, err
		}
		valTok, err := p.expect(lexer.TOKEN_BOOL_LIT)
		if err != nil {
			return nil, err
		}
		prog.ARMemory = valTok.Value == "true"
	}
	for p.peek().Type != lexer.TOKEN_EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		}
	}
	return prog, nil
}

func (p *Parser) parseStatement() (ast.Node, error) {
	t := p.peek()
	switch t.Type {
	case lexer.TOKEN_LIB:
		return p.parseLibStmt()
	case lexer.TOKEN_TYPE_INT, lexer.TOKEN_TYPE_FLOAT, lexer.TOKEN_TYPE_STRING, lexer.TOKEN_TYPE_BOOL, lexer.TOKEN_TYPE_LIST:
		return p.parseVarDecl()
	case lexer.TOKEN_IF:
		return p.parseIfStmt()
	case lexer.TOKEN_WHEN:
		return p.parseWhenStmt()
	case lexer.TOKEN_FOR:
		return p.parseForStmt()
	case lexer.TOKEN_PRINT:
		return p.parsePrintStmt()
	case lexer.TOKEN_RETURN:
		return p.parseReturnStmt()
	case lexer.TOKEN_BREAK:
		line := p.peek().Line
		p.advance()
		return &ast.BreakStmt{Line: line}, nil
	case lexer.TOKEN_FUNC:
		return p.parseFuncDecl(false)
	case lexer.TOKEN_EXPORT:
		p.advance()
		if p.peek().Type != lexer.TOKEN_FUNC {
			return nil, fmt.Errorf("line %d: expected 'func' after 'export'", p.peek().Line)
		}
		return p.parseFuncDecl(true)
	case lexer.TOKEN_FREE:
		return p.parseFreeStmt()
	case lexer.TOKEN_ALLOC:
		return p.parseAllocStmt()
	case lexer.TOKEN_IDENT:
		return p.parseIdentStmt()
	case lexer.TOKEN_EOF:
		return nil, nil
	default:
		return nil, fmt.Errorf("line %d: unexpected token %s (%q)", t.Line, t.Type, t.Value)
	}
}

func (p *Parser) parseLibStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	if p.peek().Type == lexer.TOKEN_IMPORT {
		p.pos--
		return p.parseLibImport()
	}
	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_ASSIGN); err != nil {
		return nil, err
	}
	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, ok := val.(*ast.CallExpr); !ok {
		return nil, fmt.Errorf("line %d: lib variable must be assigned a library call", line)
	}
	return &ast.VarDecl{TypeName: "lib", Name: nameTok.Value, Value: val, Line: line}, nil
}

func (p *Parser) parseLibImport() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	if _, err := p.expect(lexer.TOKEN_IMPORT); err != nil {
		return nil, err
	}
	modTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	node := &ast.LibImport{Module: modTok.Value, Line: line}
	if p.peek().Type == lexer.TOKEN_DOT {
		p.advance()
		symTok, err := p.expect(lexer.TOKEN_IDENT)
		if err != nil {
			return nil, err
		}
		node.Symbol = symTok.Value
	}
	return node, nil
}

func (p *Parser) parseVarDecl() (ast.Node, error) {
	typeTok := p.advance()
	line := typeTok.Line
	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_ASSIGN); err != nil {
		return nil, err
	}
	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	typeName := strings.ToLower(strings.TrimPrefix(string(typeTok.Type), "TYPE_"))
	return &ast.VarDecl{TypeName: typeName, Name: nameTok.Value, Value: val, Line: line}, nil
}

func (p *Parser) parseIdentStmt() (ast.Node, error) {
	line := p.peek().Line
	nameTok := p.advance()

	if p.peek().Type == lexer.TOKEN_LBRACKET {
		p.advance()
		idx, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TOKEN_RBRACKET); err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TOKEN_ASSIGN); err != nil {
			return nil, err
		}
		val, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &ast.IndexAssign{Name: nameTok.Value, Index: idx, Value: val, Line: line}, nil
	}

	if p.peek().Type == lexer.TOKEN_ASSIGN {
		p.advance()
		val, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &ast.Reassign{Name: nameTok.Value, Value: val, Line: line}, nil
	}

	p.pos--
	expr, err := p.parseIdentOrCall()
	if err != nil {
		return nil, err
	}
	return &ast.ExprStmt{Expr: expr}, nil
}

func (p *Parser) parseIfStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
		return nil, err
	}

	var body []ast.Node
	var elseIfs []*ast.ElseIfClause
	var elsebody []ast.Node

	for {
		p.skipNewlines()
		t := p.peek()
		if t.Type == lexer.TOKEN_STOP {
			p.advance()
			if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
				return nil, err
			}
			break
		}
		if t.Type == lexer.TOKEN_ELIF {
			p.advance()
			elifCond, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
				return nil, err
			}
			elifBody, err := p.parseBodyUntilStopOrElse()
			if err != nil {
				return nil, err
			}
			elseIfs = append(elseIfs, &ast.ElseIfClause{Condition: elifCond, Body: elifBody})
			continue
		}
		if t.Type == lexer.TOKEN_ELSE {
			p.advance()
			if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
				return nil, err
			}
			elsebody, err = p.parseBody()
			if err != nil {
				return nil, err
			}
			break
		}
		if t.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("line %d: unterminated if block", line)
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			body = append(body, stmt)
		}
	}
	return &ast.IfStmt{Condition: cond, Body: body, ElseIf: elseIfs, Else: elsebody, Line: line}, nil
}

func (p *Parser) parseBodyUntilStopOrElse() ([]ast.Node, error) {
	var body []ast.Node
	for {
		p.skipNewlines()
		t := p.peek()
		if t.Type == lexer.TOKEN_STOP || t.Type == lexer.TOKEN_ELSE || t.Type == lexer.TOKEN_ELIF {
			break
		}
		if t.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("line %d: unterminated block", t.Line)
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			body = append(body, stmt)
		}
	}
	return body, nil
}

func (p *Parser) parseWhenStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
		return nil, err
	}
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	return &ast.WhenStmt{Condition: cond, Body: body, Line: line}, nil
}

func (p *Parser) parseForStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	varTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_WITH); err != nil {
		return nil, err
	}
	iterExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
		return nil, err
	}
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if call, ok := iterExpr.(*ast.CallExpr); ok && call.Func == "num" && call.Module == "" {
		if len(call.Args) != 1 {
			return nil, fmt.Errorf("line %d: num() requires exactly 1 argument", line)
		}
		return &ast.ForNumStmt{Var: varTok.Value, Count: call.Args[0].Value, Body: body, Line: line}, nil
	}
	return &ast.ForEachStmt{Var: varTok.Value, Iterable: iterExpr, Body: body, Line: line}, nil
}

func (p *Parser) parsePrintStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
		return nil, err
	}
	var vals []ast.Node
	for p.peek().Type != lexer.TOKEN_RPAREN && p.peek().Type != lexer.TOKEN_EOF {
		if len(vals) > 0 {
			if p.peek().Type == lexer.TOKEN_COMMA {
				p.advance()
			}
		}
		v, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
		return nil, err
	}
	return &ast.PrintStmt{Values: vals, Line: line}, nil
}

func (p *Parser) parseReturnStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ast.ReturnStmt{Value: val, Line: line}, nil
}

func (p *Parser) parseFuncDecl(exported bool) (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
		return nil, err
	}
	var params []string
	for p.peek().Type != lexer.TOKEN_RPAREN && p.peek().Type != lexer.TOKEN_EOF {
		if len(params) > 0 {
			if _, err := p.expect(lexer.TOKEN_COMMA); err != nil {
				return nil, err
			}
		}
		pTok, err := p.expect(lexer.TOKEN_IDENT)
		if err != nil {
			return nil, err
		}
		params = append(params, pTok.Value)
	}
	if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
		return nil, err
	}
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	return &ast.FuncDecl{Name: nameTok.Value, Params: params, Body: body, Exported: exported, Line: line}, nil
}

func (p *Parser) parseFreeStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
		return nil, err
	}
	return &ast.FreeStmt{Name: nameTok.Value, Line: line}, nil
}

func (p *Parser) parseAllocStmt() (ast.Node, error) {
	line := p.peek().Line
	p.advance()
	if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_COMMA); err != nil {
		return nil, err
	}
	size, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
		return nil, err
	}
	return &ast.AllocStmt{Name: nameTok.Value, Size: size, Line: line}, nil
}

func (p *Parser) parseBody() ([]ast.Node, error) {
	var body []ast.Node
	for {
		p.skipNewlines()
		t := p.peek()
		if t.Type == lexer.TOKEN_STOP {
			p.advance()
			if _, err := p.expect(lexer.TOKEN_SEMICOLON); err != nil {
				return nil, err
			}
			break
		}
		if t.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("line %d: unterminated block, missing 'stop;'", t.Line)
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			body = append(body, stmt)
		}
	}
	return body, nil
}

func (p *Parser) parseExpr() (ast.Node, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (ast.Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == lexer.TOKEN_OR {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: "or", Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (ast.Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == lexer.TOKEN_AND {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: "and", Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseNot() (ast.Node, error) {
	if p.peek().Type == lexer.TOKEN_NOT {
		p.advance()
		operand, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "not", Operand: operand}, nil
	}
	return p.parseComparison()
}

func (p *Parser) parseComparison() (ast.Node, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.Type != lexer.TOKEN_LT && t.Type != lexer.TOKEN_LTE &&
			t.Type != lexer.TOKEN_GT && t.Type != lexer.TOKEN_GTE &&
			t.Type != lexer.TOKEN_EQ && t.Type != lexer.TOKEN_NEQ {
			break
		}
		op := p.advance().Value
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAddSub() (ast.Node, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.Type != lexer.TOKEN_PLUS && t.Type != lexer.TOKEN_MINUS {
			break
		}
		op := p.advance().Value
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMulDiv() (ast.Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.Type != lexer.TOKEN_STAR && t.Type != lexer.TOKEN_SLASH && t.Type != lexer.TOKEN_PERCENT {
			break
		}
		op := p.advance().Value
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (ast.Node, error) {
	if p.peek().Type == lexer.TOKEN_MINUS {
		p.advance()
		operand, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "-", Operand: operand}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (ast.Node, error) {
	t := p.peek()
	switch t.Type {
	case lexer.TOKEN_INT_LIT:
		p.advance()
		v, _ := strconv.ParseInt(t.Value, 10, 64)
		return &ast.IntLit{Value: v}, nil
	case lexer.TOKEN_FLOAT_LIT:
		p.advance()
		v, _ := strconv.ParseFloat(t.Value, 64)
		return &ast.FloatLit{Value: v}, nil
	case lexer.TOKEN_STRING_LIT:
		p.advance()
		return &ast.StringLit{Value: t.Value}, nil
	case lexer.TOKEN_BOOL_LIT:
		p.advance()
		return &ast.BoolLit{Value: t.Value == "true"}, nil
	case lexer.TOKEN_LBRACKET:
		return p.parseListLit()
	case lexer.TOKEN_LPAREN:
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
			return nil, err
		}
		return expr, nil
	case lexer.TOKEN_REF:
		p.advance()
		if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
			return nil, err
		}
		nameTok, err := p.expect(lexer.TOKEN_IDENT)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
			return nil, err
		}
		return &ast.RefExpr{Name: nameTok.Value, Line: t.Line}, nil
	case lexer.TOKEN_DEREF:
		p.advance()
		if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
			return nil, err
		}
		nameTok, err := p.expect(lexer.TOKEN_IDENT)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
			return nil, err
		}
		return &ast.DerefExpr{Name: nameTok.Value, Line: t.Line}, nil
	case lexer.TOKEN_IDENT:
		return p.parseIdentOrCall()
	default:
		return nil, fmt.Errorf("line %d: unexpected token in expression: %s (%q)", t.Line, t.Type, t.Value)
	}
}

func (p *Parser) parseListLit() (ast.Node, error) {
	p.advance()
	var elems []ast.Node
	for p.peek().Type != lexer.TOKEN_RBRACKET && p.peek().Type != lexer.TOKEN_EOF {
		if len(elems) > 0 {
			if _, err := p.expect(lexer.TOKEN_COMMA); err != nil {
				return nil, err
			}
		}
		v, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elems = append(elems, v)
	}
	if _, err := p.expect(lexer.TOKEN_RBRACKET); err != nil {
		return nil, err
	}
	return &ast.ListLit{Elements: elems}, nil
}

func (p *Parser) parseIdentOrCall() (ast.Node, error) {
	nameTok := p.advance()
	line := nameTok.Line
	if p.peek().Type == lexer.TOKEN_DOT {
		p.advance()
		funcTok, err := p.expect(lexer.TOKEN_IDENT)
		if err != nil {
			return nil, err
		}
		if p.peek().Type == lexer.TOKEN_LPAREN {
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			base := &ast.CallExpr{Module: nameTok.Value, Func: funcTok.Value, Args: args, Line: line}
			return p.parsePostfix(base)
		}
		return &ast.Identifier{Name: nameTok.Value + "." + funcTok.Value, Line: line}, nil
	}
	if p.peek().Type == lexer.TOKEN_LPAREN {
		args, err := p.parseCallArgs()
		if err != nil {
			return nil, err
		}
		base := &ast.CallExpr{Func: nameTok.Value, Args: args, Line: line}
		return p.parsePostfix(base)
	}
	base := &ast.Identifier{Name: nameTok.Value, Line: line}
	return p.parsePostfix(base)
}

func (p *Parser) parsePostfix(base ast.Node) (ast.Node, error) {
	for p.peek().Type == lexer.TOKEN_LBRACKET {
		line := p.peek().Line
		p.advance()
		idx, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TOKEN_RBRACKET); err != nil {
			return nil, err
		}
		base = &ast.IndexExpr{List: base, Index: idx, Line: line}
	}
	return base, nil
}

func (p *Parser) parseCallArgs() ([]ast.CallArg, error) {
	if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
		return nil, err
	}
	var args []ast.CallArg
	for p.peek().Type != lexer.TOKEN_RPAREN && p.peek().Type != lexer.TOKEN_EOF {
		if len(args) > 0 {
			if _, err := p.expect(lexer.TOKEN_COMMA); err != nil {
				return nil, err
			}
		}
		if p.peek().Type == lexer.TOKEN_IDENT {
			saved := p.pos
			nameTok := p.advance()
			if p.peek().Type == lexer.TOKEN_ASSIGN {
				p.advance()
				val, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, ast.CallArg{Name: nameTok.Value, Value: val})
				continue
			}
			p.pos = saved
		}
		val, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, ast.CallArg{Value: val})
	}
	if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
		return nil, err
	}
	return args, nil
}
