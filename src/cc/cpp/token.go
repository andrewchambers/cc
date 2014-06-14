package cpp

import (
	"fmt"
)

const (
	TOK_FOR = 1000
	TOK_WHILE
	TOK_DO
	TOK_IF
	TOK_GOTO
	TOK_STRUCT
	TOK_SIGNED
	TOK_UNSIGNED
	TOK_TYPEDEF
	TOK_RETURN
	TOK_INT
	TOK_VOID
	TOK_SIZEOF
	TOK_IDENTIFIER
	TOK_CONSTANT_INT
	TOK_INC_OP
	TOK_PTR_OP
	TOK_OR_OP
	TOK_AND_OP
	TOK_EQ_OP
)

type FilePos struct {
	File string
	Line int
	Col  int
}

func (pos FilePos) String() string {
	return fmt.Sprintf("Position line %d col %d of %s.", pos.Line, pos.Col, pos.File)
}

type TokenKind int

func (tk TokenKind) String() string {
	switch tk {
	case TOK_IDENTIFIER:
		return "TOK_IDENTIFIER"
	case TOK_FOR:
		return "TOK_FOR"
	case TOK_INT:
		return "TOK_INT"
	default:
		return fmt.Sprintf("TOK %c", (int)tk)
	}
}

//Token represents a grouping of characters
//that provide semantic meaning in a C program.
type Token struct {
	Kind TokenKind
	Val  string
	Pos  FilePos
}

func (t Token) String() string {
	return fmt.Sprintf("%s %s at %s", t.Kind, t.Val, t.Pos)
}

var keywordLUT = map[string]TokenKind{
	"for":      TOK_FOR,
	"while":    TOK_WHILE,
	"do":       TOK_DO,
	"if":       TOK_IF,
	"goto":     TOK_GOTO,
	"struct":   TOK_STRUCT,
	"signed":   TOK_SIGNED,
	"unsigned": TOK_UNSIGNED,
	"typedef":  TOK_TYPEDEF,
	"return":   TOK_RETURN,
	"int":      TOK_INT,
	"void":     TOK_VOID,
	"sizeof":   TOK_SIZEOF,
}
