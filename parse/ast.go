package parse

import "github.com/andrewchambers/cc/cpp"

type Node interface {
	GetType() CType
}

type Constant struct {
	Val  int64
	Pos  cpp.FilePos
	Type CType
}

func (c *Constant) GetType() CType { return c.Type }

type Return struct {
	Pos  cpp.FilePos
	Expr Node
}

func (r *Return) GetType() CType { return nil }

type Unop struct {
	Op      cpp.TokenKind
	Pos     cpp.FilePos
	Operand Node
	Type    CType
}

func (u *Unop) GetType() CType { return u.Type }

type Binop struct {
	Op   cpp.TokenKind
	Pos  cpp.FilePos
	L    Node
	R    Node
	Type CType
}

func (b *Binop) GetType() CType { return b.Type }

type Function struct {
	Name     string
	Pos      cpp.FilePos
	FuncType *FunctionType
	Body     []Node
}

func (f *Function) GetType() CType { return f.FuncType }

type DeclList struct {
	Symbols []Symbol
}

func (d *DeclList) GetType() CType { return nil }

type Ident struct {
	Sym Symbol
}

func (f *Ident) GetType() CType {
	switch sym := f.Sym.(type) {
	case *GSymbol:
		return sym.Type
	}
	panic("unimplemented")
}
