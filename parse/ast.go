package parse

import "github.com/andrewchambers/cc/cpp"

type Node interface{}

type Constant struct {
	Val  int64
	Pos  cpp.FilePos
	Type CType
}

type Return struct {
	Pos  cpp.FilePos
	Expr Node
}

type Binop struct {
	Op  cpp.TokenKind
	Pos cpp.FilePos
	L   Node
	R   Node
}

type Function struct {
	Name     string
	Pos      cpp.FilePos
	FuncType *FunctionType
	Body     []Node
}
