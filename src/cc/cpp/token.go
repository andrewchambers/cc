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
