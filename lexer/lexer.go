package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType string

const (
	TOKEN_INT_LIT    TokenType = "INT_LIT"
	TOKEN_FLOAT_LIT  TokenType = "FLOAT_LIT"
	TOKEN_STRING_LIT TokenType = "STRING_LIT"
	TOKEN_BOOL_LIT   TokenType = "BOOL_LIT"

	TOKEN_TYPE_INT    TokenType = "TYPE_INT"
	TOKEN_TYPE_FLOAT  TokenType = "TYPE_FLOAT"
	TOKEN_TYPE_STRING TokenType = "TYPE_STRING"
	TOKEN_TYPE_BOOL   TokenType = "TYPE_BOOL"
	TOKEN_TYPE_LIST   TokenType = "TYPE_LIST"

	TOKEN_IF     TokenType = "IF"
	TOKEN_STOP   TokenType = "STOP"
	TOKEN_LIB    TokenType = "LIB"
	TOKEN_IMPORT TokenType = "IMPORT"
	TOKEN_PRINT  TokenType = "PRINT"
	TOKEN_WHEN   TokenType = "WHEN"
	TOKEN_FOR    TokenType = "FOR"
	TOKEN_WITH   TokenType = "WITH"
	TOKEN_FUNC   TokenType = "FUNC"
	TOKEN_EXPORT TokenType = "EXPORT"
	TOKEN_RETURN TokenType = "RETURN"
	TOKEN_BREAK  TokenType = "BREAK"
	TOKEN_ELSE   TokenType = "ELSE"
	TOKEN_ELIF   TokenType = "ELIF"

	TOKEN_ARMEMORY TokenType = "ARMEMORY"
	TOKEN_FREE     TokenType = "FREE"
	TOKEN_ALLOC    TokenType = "ALLOC"
	TOKEN_REF      TokenType = "REF"
	TOKEN_DEREF    TokenType = "DEREF"

	TOKEN_IDENT TokenType = "IDENT"

	TOKEN_ASSIGN  TokenType = "ASSIGN"
	TOKEN_PLUS    TokenType = "PLUS"
	TOKEN_MINUS   TokenType = "MINUS"
	TOKEN_STAR    TokenType = "STAR"
	TOKEN_SLASH   TokenType = "SLASH"
	TOKEN_PERCENT TokenType = "PERCENT"
	TOKEN_EQ      TokenType = "EQ"
	TOKEN_NEQ     TokenType = "NEQ"
	TOKEN_LT      TokenType = "LT"
	TOKEN_LTE     TokenType = "LTE"
	TOKEN_GT      TokenType = "GT"
	TOKEN_GTE     TokenType = "GTE"
	TOKEN_AND     TokenType = "AND"
	TOKEN_OR      TokenType = "OR"
	TOKEN_NOT     TokenType = "NOT"

	TOKEN_LPAREN    TokenType = "LPAREN"
	TOKEN_RPAREN    TokenType = "RPAREN"
	TOKEN_LBRACKET  TokenType = "LBRACKET"
	TOKEN_RBRACKET  TokenType = "RBRACKET"
	TOKEN_SEMICOLON TokenType = "SEMICOLON"
	TOKEN_DOT       TokenType = "DOT"
	TOKEN_COMMA     TokenType = "COMMA"

	TOKEN_EOF     TokenType = "EOF"
	TOKEN_NEWLINE TokenType = "NEWLINE"
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, line=%d)", t.Type, t.Value, t.Line)
}

var keywords = map[string]TokenType{
	"int":      TOKEN_TYPE_INT,
	"float":    TOKEN_TYPE_FLOAT,
	"string":   TOKEN_TYPE_STRING,
	"bool":     TOKEN_TYPE_BOOL,
	"list":     TOKEN_TYPE_LIST,
	"if":       TOKEN_IF,
	"stop":     TOKEN_STOP,
	"lib":      TOKEN_LIB,
	"import":   TOKEN_IMPORT,
	"print":    TOKEN_PRINT,
	"when":     TOKEN_WHEN,
	"for":      TOKEN_FOR,
	"with":     TOKEN_WITH,
	"func":     TOKEN_FUNC,
	"export":   TOKEN_EXPORT,
	"return":   TOKEN_RETURN,
	"break":    TOKEN_BREAK,
	"else":     TOKEN_ELSE,
	"elif":     TOKEN_ELIF,
	"true":     TOKEN_BOOL_LIT,
	"false":    TOKEN_BOOL_LIT,
	"armemory": TOKEN_ARMEMORY,
	"free":     TOKEN_FREE,
	"alloc":    TOKEN_ALLOC,
	"ref":      TOKEN_REF,
	"deref":    TOKEN_DEREF,
	"and":      TOKEN_AND,
	"or":       TOKEN_OR,
	"not":      TOKEN_NOT,
}

type Lexer struct {
	src  []rune
	pos  int
	line int
}

func New(src string) *Lexer {
	return &Lexer{src: []rune(src), pos: 0, line: 1}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) peekAt(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *Lexer) advance() rune {
	ch := l.src[l.pos]
	l.pos++
	return ch
}

func (l *Lexer) skipComment() {
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) readString() Token {
	line := l.line
	l.pos++
	var sb strings.Builder
	for l.pos < len(l.src) && l.src[l.pos] != '"' {
		if l.src[l.pos] == '\\' && l.pos+1 < len(l.src) {
			l.pos++
			switch l.src[l.pos] {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '"':
				sb.WriteRune('"')
			case '{':
				sb.WriteRune('{')
			default:
				sb.WriteRune(l.src[l.pos])
			}
			l.pos++
			continue
		}
		if l.src[l.pos] == '{' {
			l.pos++
			var varName strings.Builder
			for l.pos < len(l.src) && l.src[l.pos] != '}' {
				varName.WriteRune(l.src[l.pos])
				l.pos++
			}
			l.pos++
			sb.WriteString("\x00INTERP\x00" + varName.String() + "\x00ENDINTERP\x00")
			continue
		}
		sb.WriteRune(l.advance())
	}
	l.pos++
	return Token{Type: TOKEN_STRING_LIT, Value: sb.String(), Line: line}
}

func (l *Lexer) readNumber() Token {
	line := l.line
	var sb strings.Builder
	isFloat := false
	for l.pos < len(l.src) && (unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '.') {
		if l.src[l.pos] == '.' {
			isFloat = true
		}
		sb.WriteRune(l.advance())
	}
	if isFloat {
		return Token{Type: TOKEN_FLOAT_LIT, Value: sb.String(), Line: line}
	}
	return Token{Type: TOKEN_INT_LIT, Value: sb.String(), Line: line}
}

func (l *Lexer) readIdent() Token {
	line := l.line
	var sb strings.Builder
	for l.pos < len(l.src) && (unicode.IsLetter(l.src[l.pos]) || unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '_') {
		sb.WriteRune(l.advance())
	}
	val := sb.String()
	if tt, ok := keywords[val]; ok {
		return Token{Type: tt, Value: val, Line: line}
	}
	return Token{Type: TOKEN_IDENT, Value: val, Line: line}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token

	for l.pos < len(l.src) {
		ch := l.peek()

		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}

		if ch == '\n' {
			tokens = append(tokens, Token{Type: TOKEN_NEWLINE, Value: "\\n", Line: l.line})
			l.line++
			l.advance()
			continue
		}

		if ch == '$' && l.peekAt(1) == '/' {
			l.skipComment()
			continue
		}

		if ch == '"' {
			tokens = append(tokens, l.readString())
			continue
		}

		if unicode.IsDigit(ch) {
			tokens = append(tokens, l.readNumber())
			continue
		}

		if unicode.IsLetter(ch) || ch == '_' {
			tokens = append(tokens, l.readIdent())
			continue
		}

		line := l.line

		if ch == '<' && l.peekAt(1) == '=' {
			tokens = append(tokens, Token{Type: TOKEN_LTE, Value: "<=", Line: line})
			l.pos += 2
			continue
		}
		if ch == '>' && l.peekAt(1) == '=' {
			tokens = append(tokens, Token{Type: TOKEN_GTE, Value: ">=", Line: line})
			l.pos += 2
			continue
		}
		if ch == '!' && l.peekAt(1) == '=' {
			tokens = append(tokens, Token{Type: TOKEN_NEQ, Value: "!=", Line: line})
			l.pos += 2
			continue
		}
		if ch == '=' && l.peekAt(1) == '=' {
			tokens = append(tokens, Token{Type: TOKEN_EQ, Value: "==", Line: line})
			l.pos += 2
			continue
		}

		switch ch {
		case '=':
			tokens = append(tokens, Token{Type: TOKEN_ASSIGN, Value: "=", Line: line})
		case '+':
			tokens = append(tokens, Token{Type: TOKEN_PLUS, Value: "+", Line: line})
		case '-':
			tokens = append(tokens, Token{Type: TOKEN_MINUS, Value: "-", Line: line})
		case '*':
			tokens = append(tokens, Token{Type: TOKEN_STAR, Value: "*", Line: line})
		case '/':
			tokens = append(tokens, Token{Type: TOKEN_SLASH, Value: "/", Line: line})
		case '%':
			tokens = append(tokens, Token{Type: TOKEN_PERCENT, Value: "%", Line: line})
		case '<':
			tokens = append(tokens, Token{Type: TOKEN_LT, Value: "<", Line: line})
		case '>':
			tokens = append(tokens, Token{Type: TOKEN_GT, Value: ">", Line: line})
		case '(':
			tokens = append(tokens, Token{Type: TOKEN_LPAREN, Value: "(", Line: line})
		case ')':
			tokens = append(tokens, Token{Type: TOKEN_RPAREN, Value: ")", Line: line})
		case '[':
			tokens = append(tokens, Token{Type: TOKEN_LBRACKET, Value: "[", Line: line})
		case ']':
			tokens = append(tokens, Token{Type: TOKEN_RBRACKET, Value: "]", Line: line})
		case ';':
			tokens = append(tokens, Token{Type: TOKEN_SEMICOLON, Value: ";", Line: line})
		case '.':
			tokens = append(tokens, Token{Type: TOKEN_DOT, Value: ".", Line: line})
		case ',':
			tokens = append(tokens, Token{Type: TOKEN_COMMA, Value: ",", Line: line})
		default:
			return nil, fmt.Errorf("line %d: unexpected character %q", l.line, ch)
		}
		l.advance()
	}

	tokens = append(tokens, Token{Type: TOKEN_EOF, Value: "", Line: l.line})
	return tokens, nil
}
