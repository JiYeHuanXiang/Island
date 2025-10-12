package parser

import (
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input string
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
	}
}

var keywords = map[string]int{
	"d":   D,
	"h":   H,
	"l":   L,
	"f":   F,
	"df":  F, // df等同于f
	"a":   A,
	"c":   C,
	"p":   P,
	"b":   B,
	"k":   K,
	"q":   Q,
	"m":   M,
	"max": MAX,
	"min": MIN,
	"kh":  KH,
	"kl":  KL,
}

func (l *Lexer) Lex(lval *yySymType) int {
	if l.pos >= len(l.input) {
		return 0
	}

	r, width := utf8.DecodeRuneInString(l.input[l.pos:])

	// 处理空格
	if unicode.IsSpace(r) {
		l.pos += width
		return l.Lex(lval)
	}

	// 处理数字
	if unicode.IsDigit(r) {
		token := l.lexNumber(lval)
		fmt.Printf("识别到数字: %d [token=%d]\n", lval.Num, token)
		return token
	}

	// 首先检查df操作符
	if l.pos+1 < len(l.input) && l.input[l.pos:l.pos+2] == "df" {
		l.pos += 2
		fmt.Printf("识别到df运算符 (等同于f) [token=%d]\n", F)
		return F
	}

	// 处理d操作符
	if r == 'd' || r == 'D' {
		l.pos += width
		fmt.Printf("识别到d运算符 [token=%d]\n", D)
		return D
	}

	// 检查复合操作符
	if l.pos+1 < len(l.input) {
		switch l.input[l.pos : l.pos+2] {
		case "kh":
			l.pos += 2
			return KH
		case "kl":
			l.pos += 2
			return KL
		}
	}

	switch {
	case isIdentStart(r):
		return l.lexIdent(lval)
	case r == 'h':
		l.pos += width
		return H
	case r == 'l':
		l.pos += width
		return L
	case r == 'f':
		l.pos += width
		return F
	case r == 'a':
		l.pos += width
		return A
	case r == 'c':
		l.pos += width
		return C
	case r == 'p':
		l.pos += width
		return P
	case r == 'b':
		l.pos += width
		return B
	case r == 'k':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == 'h' {
			l.pos += 2
			return KH
		} else if l.pos+1 < len(l.input) && l.input[l.pos+1] == 'l' {
			l.pos += 2
			return KL
		}
		l.pos += width
		return K
	case r == 'q':
		l.pos += width
		return Q
	case r == 'm':
		l.pos += width
		return M
	case r == '+':
		l.pos += width
		return ADD
	case r == '-':
		l.pos += width
		return SUB
	case r == '*':
		l.pos += width
		return MUL
	case r == '/':
		l.pos += width
		return DIV
	case r == '(':
		l.pos += width
		return LPAREN
	case r == ')':
		l.pos += width
		return RPAREN
	case r == ',':
		l.pos += width
		return COMMA
	case r == '?':
		l.pos += width
		return QUESTION
	case r == ':':
		l.pos += width
		return COLON
	case r == '[':
		l.pos += width
		return LBRACKET
	case r == ']':
		l.pos += width
		return RBRACKET
	case r == '^':
		l.pos += width
		return CIRCUMFLEX
	case r == '%':
		l.pos += width
		return MOD
	case r == '=':
		l.pos += width
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos += width
			return EQ
		}
		return ASSIGN
	case r == '!':
		l.pos += width
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos += width
			return NEQ
		}
		return int('!')
	case r == '>':
		l.pos += width
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos += width
			return GE
		}
		return GT
	case r == '<':
		l.pos += width
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos += width
			return LE
		}
		return LT
	case r == '&':
		l.pos += width
		return BITAND
	case r == '|':
		l.pos += width
		return BITOR
	default:
		// 检查特殊关键字
		if l.pos+3 <= len(l.input) && l.input[l.pos:l.pos+3] == "max" {
			l.pos += 3
			return MAX
		} else if l.pos+3 <= len(l.input) && l.input[l.pos:l.pos+3] == "min" {
			l.pos += 3
			return MIN
		} else if l.pos+2 <= len(l.input) && l.input[l.pos:l.pos+2] == "df" {
			l.pos += 2
			return F // df 等同于 f
		}

		l.pos += width
		return int(r)
	}
	return 0
}

func (l *Lexer) lexNumber(lval *yySymType) int {
	start := l.pos
	for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
		_, width := utf8.DecodeRuneInString(l.input[l.pos:])
		l.pos += width
	}
	num, _ := strconv.Atoi(l.input[start:l.pos])
	lval.Num = num
	return NUMBER
}

func (l *Lexer) lexIdent(lval *yySymType) int {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(rune(l.input[l.pos])) {
		l.pos++
	}
	ident := l.input[start:l.pos]

	lval.Str = ident
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

func isIdentStart(r rune) bool {
	if r == 0 {
		return false
	}
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	if r == 0 {
		return false
	}
	return isIdentStart(r) || unicode.IsDigit(r)
}

func (l *Lexer) Error(s string) {
	fmt.Printf("语法错误: %s\n", s)
}
