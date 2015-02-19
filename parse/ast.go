package parse

import "github.com/andrewchambers/cc/cpp"

type Node interface {
	GetType() CType
	GetPos() cpp.FilePos
}

type Constant struct {
	Val  int64
	Pos  cpp.FilePos
	Type CType
}

func (c *Constant) GetType() CType      { return c.Type }
func (c *Constant) GetPos() cpp.FilePos { return c.Pos }

type Return struct {
	Pos  cpp.FilePos
	Expr Node
}

func (r *Return) GetType() CType      { return nil }
func (r *Return) GetPos() cpp.FilePos { return r.Pos }

type Index struct {
	Pos  cpp.FilePos
	Arr  Node
	Idx  Node
	Type CType
}

func (i *Index) GetType() CType      { return i.Type }
func (i *Index) GetPos() cpp.FilePos { return i.Pos }

type Unop struct {
	Op      cpp.TokenKind
	Pos     cpp.FilePos
	Operand Node
	Type    CType
}

func (u *Unop) GetType() CType      { return u.Type }
func (u *Unop) GetPos() cpp.FilePos { return u.Pos }

type Binop struct {
	Op   cpp.TokenKind
	Pos  cpp.FilePos
	L    Node
	R    Node
	Type CType
}

func (b *Binop) GetType() CType      { return b.Type }
func (b *Binop) GetPos() cpp.FilePos { return b.Pos }

type Function struct {
	Name     string
	Pos      cpp.FilePos
	FuncType *FunctionType
	Body     []Node
}

func (f *Function) GetType() CType      { return f.FuncType }
func (f *Function) GetPos() cpp.FilePos { return f.Pos }

type DeclList struct {
	Pos         cpp.FilePos
	Symbols     []Symbol
	Inits       []Node
	FoldedInits []*FoldedConstant
}

func (d *DeclList) GetType() CType      { return nil }
func (d *DeclList) GetPos() cpp.FilePos { return d.Pos }

type Ident struct {
	Pos cpp.FilePos
	Sym Symbol
}

func (i *Ident) GetType() CType {
	switch sym := i.Sym.(type) {
	case *GSymbol:
		return sym.Type
	}
	panic("unimplemented")
}
func (i *Ident) GetPos() cpp.FilePos { return i.Pos }
