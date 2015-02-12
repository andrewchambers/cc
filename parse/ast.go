package parse

import "github.com/andrewchambers/cc/cpp"

type Node interface{}

type Binop struct {
	Op  cpp.TokenKind
	Pos cpp.FilePos
	L   Node
	R   Node
}
