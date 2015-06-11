package parse

import "github.com/andrewchambers/cc/cpp"

type Node interface {
	GetPos() cpp.FilePos
}

type Expr interface {
	Node
	GetType() CType
}

type TranslationUnit struct {
	TopLevels      []Node
	AnonymousInits []Node
}

type Constant struct {
	Val  int64
	Pos  cpp.FilePos
	Type CType
}

func (c *Constant) GetType() CType      { return c.Type }
func (c *Constant) GetPos() cpp.FilePos { return c.Pos }

type Initializer struct {
	Pos   cpp.FilePos
	Inits []Node
}

func (i *Initializer) GetPos() cpp.FilePos { return i.Pos }

type Return struct {
	Pos cpp.FilePos
	Ret Expr
}

func (r *Return) GetPos() cpp.FilePos { return r.Pos }

type Index struct {
	Pos  cpp.FilePos
	Arr  Node
	Idx  Node
	Type CType
}

func (i *Index) GetType() CType      { return i.Type }
func (i *Index) GetPos() cpp.FilePos { return i.Pos }

type Cast struct {
	Pos     cpp.FilePos
	Operand Expr
	Type    CType
}

func (c *Cast) GetType() CType      { return c.Type }
func (c *Cast) GetPos() cpp.FilePos { return c.Pos }

type Block struct {
	Pos  cpp.FilePos
	Body []Node
}

func (b *Block) GetPos() cpp.FilePos { return b.Pos }

type EmptyStmt struct {
	Pos cpp.FilePos
}

func (e *EmptyStmt) GetPos() cpp.FilePos { return e.Pos }

type ExprStmt struct {
	Pos  cpp.FilePos
	Expr Expr
}

func (e *ExprStmt) GetPos() cpp.FilePos { return e.Pos }

type Goto struct {
	IsBreak bool
	IsCont  bool
	Pos     cpp.FilePos
	Label   string
}

func (g *Goto) GetPos() cpp.FilePos { return g.Pos }

type LabeledStmt struct {
	Pos       cpp.FilePos
	AnonLabel string
	Label     string
	Stmt      Node
	IsCase    bool
	IsDefault bool
}

func (l *LabeledStmt) GetPos() cpp.FilePos { return l.Pos }

type If struct {
	Pos   cpp.FilePos
	Cond  Node
	Stmt  Node
	Else  Node
	LElse string
}

func (i *If) GetPos() cpp.FilePos { return i.Pos }

type SwitchCase struct {
	V     int64
	Label string
}

type Switch struct {
	Pos      cpp.FilePos
	Expr     Node
	Stmt     Node
	Cases    []SwitchCase
	LDefault string
	LAfter   string
}

func (sw *Switch) GetPos() cpp.FilePos { return sw.Pos }

type For struct {
	Pos    cpp.FilePos
	Init   Node
	Cond   Node
	Step   Node
	Body   Node
	LStart string
	LEnd   string
}

func (f *For) GetPos() cpp.FilePos { return f.Pos }

type While struct {
	Pos    cpp.FilePos
	Cond   Node
	Body   Node
	LStart string
	LEnd   string
}

func (w *While) GetPos() cpp.FilePos { return w.Pos }

type DoWhile struct {
	Pos    cpp.FilePos
	Cond   Node
	Body   Node
	LStart string
	LCond  string
	LEnd   string
}

func (d *DoWhile) GetPos() cpp.FilePos { return d.Pos }

type Unop struct {
	Op      cpp.TokenKind
	Pos     cpp.FilePos
	Operand Node
	Type    CType
}

func (u *Unop) GetType() CType      { return u.Type }
func (u *Unop) GetPos() cpp.FilePos { return u.Pos }

type Selector struct {
	Op      cpp.TokenKind
	Pos     cpp.FilePos
	Type    CType
	Operand Expr
	Sel     string
}

func (s *Selector) GetType() CType      { return s.Type }
func (s *Selector) GetPos() cpp.FilePos { return s.Pos }

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
	Name         string
	Pos          cpp.FilePos
	FuncType     *FunctionType
	ParamSymbols []*LSymbol
	Body         []Node
}

func (f *Function) GetType() CType      { return f.FuncType }
func (f *Function) GetPos() cpp.FilePos { return f.Pos }

type Call struct {
	Pos      cpp.FilePos
	FuncLike Expr
	Args     []Expr
	Type     CType
}

func (c *Call) GetType() CType      { return c.Type }
func (c *Call) GetPos() cpp.FilePos { return c.Pos }

type DeclList struct {
	Pos     cpp.FilePos
	Storage SClass
	Symbols []Symbol
	Inits   []Expr
}

func (d *DeclList) GetPos() cpp.FilePos { return d.Pos }

type String struct {
	Pos   cpp.FilePos
	Val   string
	Label string
}

func (s *String) GetType() CType      { return &Ptr{CChar} }
func (s *String) GetPos() cpp.FilePos { return s.Pos }

type Ident struct {
	Pos cpp.FilePos
	Sym Symbol
}

func (i *Ident) GetType() CType {
	switch sym := i.Sym.(type) {
	case *GSymbol:
		return sym.Type
	case *LSymbol:
		return sym.Type
	}
	panic("unimplemented")
}

func (i *Ident) GetPos() cpp.FilePos { return i.Pos }
