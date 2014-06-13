package cpp

type FilePos struct {
	File string
	Line int
	Col  int
}

type TokenKind int

//Token represents a grouping of characters
//that provide semantic meaning in a C program.
type Token struct {
	Kind TokenKind
	Val  string
	Pos  FilePos
}

const (
	TOK_FOR = iota
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
)

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
