package cpp

import (
	"fmt"
)

// The list of tokens.
const (

	// Single char tokens are themselves.
	ADD       = '+'
	SUB       = '-'
	MUL       = '*'
	QUO       = '/'
	REM       = '%'
	AND       = '&'
	OR        = '|'
	XOR       = '^'
	QUESTION  = '?'
	HASH      = '#'
	LSS       = '<'
	GTR       = '>'
	ASSIGN    = '='
	NOT       = '!'
	BNOT      = '~'
	LPAREN    = '('
	LBRACK    = '['
	LBRACE    = '{'
	COMMA     = ','
	PERIOD    = '.'
	RPAREN    = ')'
	RBRACK    = ']'
	RBRACE    = '}'
	SEMICOLON = ';'
	COLON     = ':'

	ERROR = 10000 + iota
	EOF
	//some cpp only tokens
	FUNCLIKE_DEFINE //Occurs after ident before paren #define ident(
	DIRECTIVE       //#if #include etc
	END_DIRECTIVE   //New line at the end of a directive
	HEADER
	// Identifiers and basic type literals
	// (these tokens stand for classes of literals)
	TYPENAME       // Same as ident, but typedefed.
	IDENT          // main
	INT_CONSTANT   // 12345
	FLOAT_CONSTANT // 123.45
	CHAR_CONSTANT  // 'a'
	STRING         // "abc"

	SHL        // <<
	SHR        // >>
	ADD_ASSIGN // +=
	SUB_ASSIGN // -=
	MUL_ASSIGN // *=
	QUO_ASSIGN // /=
	REM_ASSIGN // %=
	AND_ASSIGN // &=
	OR_ASSIGN  // |=
	XOR_ASSIGN // ^=
	SHL_ASSIGN // <<=
	SHR_ASSIGN // >>=
	LAND       // &&
	LOR        // ||
	ARROW      // ->
	INC        // ++
	DEC        // --
	EQL        // ==
	NEQ        // !=
	LEQ        // <=
	GEQ        // >=
	ELLIPSIS   // ...

	// Keywords
	REGISTER
	EXTERN
	STATIC
	SHORT
	BREAK
	CASE
	DO
	CONST
	CONTINUE
	DEFAULT
	ELSE
	FOR
	WHILE
	GOTO
	IF
	RETURN
	STRUCT
	UNION
	VOLATILE
	SWITCH
	TYPEDEF
	SIZEOF
	VOID
	CHAR
	INT
	FLOAT
	DOUBLE
	SIGNED
	UNSIGNED
	LONG
)

var tokenKindToStr = [...]string{
	HASH:            "#",
	EOF:             "EOF",
	FUNCLIKE_DEFINE: "funclikedefine",
	DIRECTIVE:       "cppdirective",
	END_DIRECTIVE:   "enddirective",
	HEADER:          "header",
	CHAR_CONSTANT:   "charconst",
	INT_CONSTANT:    "intconst",
	FLOAT_CONSTANT:  "floatconst",
	IDENT:           "ident",
	VOID:            "void",
	INT:             "int",
	LONG:            "long",
	SIGNED:          "signed",
	UNSIGNED:        "unsigned",
	FLOAT:           "float",
	DOUBLE:          "double",
	CHAR:            "char",
	STRING:          "string",
	ADD:             "'+'",
	SUB:             "'-'",
	MUL:             "'*'",
	QUO:             "'/'",
	REM:             "'%'",
	AND:             "'&'",
	OR:              "'|'",
	XOR:             "'^'",
	SHL:             "'<<'",
	SHR:             "'>>'",
	ADD_ASSIGN:      "'+='",
	SUB_ASSIGN:      "'-='",
	MUL_ASSIGN:      "'*='",
	QUO_ASSIGN:      "'/='",
	REM_ASSIGN:      "'%='",
	AND_ASSIGN:      "'&='",
	OR_ASSIGN:       "'|='",
	XOR_ASSIGN:      "'^='",
	SHL_ASSIGN:      "'<<='",
	SHR_ASSIGN:      "'>>='",
	LAND:            "'&&'",
	LOR:             "'||'",
	ARROW:           "'->'",
	INC:             "'++'",
	DEC:             "'--'",
	EQL:             "'=='",
	LSS:             "'<'",
	GTR:             "'>'",
	ASSIGN:          "'='",
	NOT:             "'!'",
	BNOT:            "'~'",
	NEQ:             "'!='",
	LEQ:             "'<='",
	GEQ:             "'>='",
	ELLIPSIS:        "'...'",
	LPAREN:          "'('",
	LBRACK:          "'['",
	LBRACE:          "'{'",
	COMMA:           "','",
	PERIOD:          "'.'",
	RPAREN:          "')'",
	RBRACK:          "']'",
	RBRACE:          "'}'",
	SEMICOLON:       "';'",
	COLON:           "':'",
	QUESTION:        "'?'",
	SIZEOF:          "sizeof",
	TYPEDEF:         "typedef",
	BREAK:           "break",
	CASE:            "case",
	CONST:           "const",
	CONTINUE:        "continue",
	DEFAULT:         "default",
	ELSE:            "else",
	FOR:             "for",
	DO:              "do",
	WHILE:           "while",
	GOTO:            "goto",
	IF:              "if",
	RETURN:          "return",
	STRUCT:          "struct",
	SWITCH:          "switch",
	STATIC:          "static",
}

var keywordLUT = map[string]TokenKind{
	"for":      FOR,
	"while":    WHILE,
	"do":       DO,
	"if":       IF,
	"else":     ELSE,
	"goto":     GOTO,
	"break":    BREAK,
	"continue": CONTINUE,
	"case":     CASE,
	"default":  DEFAULT,
	"switch":   SWITCH,
	"struct":   STRUCT,
	"signed":   SIGNED,
	"unsigned": UNSIGNED,
	"typedef":  TYPEDEF,
	"return":   RETURN,
	"void":     VOID,
	"char":     CHAR,
	"int":      INT,
	"short":    SHORT,
	"long":     LONG,
	"float":    FLOAT,
	"double":   DOUBLE,
	"sizeof":   SIZEOF,
	"static":   STATIC,
}

type TokenKind uint32

func (tk TokenKind) String() string {
	if uint32(tk) >= uint32(len(tokenKindToStr)) {
		return "Unknown"
	}
	ret := tokenKindToStr[tk]
	if ret == "" {
		return "Unknown"
	}
	return ret
}

type FilePos struct {
	File string
	Line int
	Col  int
}

func (pos FilePos) String() string {
	return fmt.Sprintf("%s:%d:%d", pos.File, pos.Line, pos.Col)
}

//Token represents a grouping of characters
//that provide semantic meaning in a C program.
type Token struct {
	Kind             TokenKind
	Val              string
	Pos              FilePos
	WasMacroExpanded bool
	hs               *hideset
}

func (t *Token) copy() *Token {
	ret := *t
	return &ret
}

func (t Token) String() string {
	if t.WasMacroExpanded {
		fmt.Sprintf("%s expanded from macro at %s", t.Val, t.Pos)
	}
	return fmt.Sprintf("%s at %s", t.Val, t.Pos)
}
