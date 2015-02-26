package parse

import "github.com/andrewchambers/cc/cpp"

type Node interface {
	GetPos() cpp.FilePos
	Children() []Node
}

type Expr interface {
	Node
	GetType() CType
}

type Constant struct {
	Val  int64
	Pos  cpp.FilePos
	Type CType
}

func (c *Constant) GetType() CType      { return c.Type }
func (c *Constant) GetPos() cpp.FilePos { return c.Pos }
func (c *Constant) Children() []Node    { return []Node{} }

type Return struct {
	Pos cpp.FilePos
	Ret Expr
}

func (r *Return) GetPos() cpp.FilePos { return r.Pos }
func (r *Return) Children() []Node    { return []Node{r.Ret} }

type Index struct {
	Pos  cpp.FilePos
	Arr  Node
	Idx  Node
	Type CType
}

func (i *Index) GetType() CType      { return i.Type }
func (i *Index) GetPos() cpp.FilePos { return i.Pos }
func (i *Index) Children() []Node    { return []Node{i.Arr, i.Idx} }

type Cast struct {
	Pos     cpp.FilePos
	Operand Node
	Type    CType
}

func (c *Cast) GetType() CType      { return c.Type }
func (c *Cast) GetPos() cpp.FilePos { return c.Pos }
func (c *Cast) Children() []Node    { return []Node{c.Operand} }

type CompndStmt struct {
	Pos  cpp.FilePos
	Body []Node
}

func (c *CompndStmt) GetPos() cpp.FilePos { return c.Pos }
func (c *CompndStmt) Children() []Node    { return c.Body }

type EmptyStmt struct {
	Pos cpp.FilePos
}

func (e *EmptyStmt) GetPos() cpp.FilePos { return e.Pos }
func (*EmptyStmt) Children() []Node      { return []Node{} }

type ExprStmt struct {
	Pos  cpp.FilePos
	Expr Expr
}

func (e *ExprStmt) GetPos() cpp.FilePos { return e.Pos }
func (e *ExprStmt) Children() []Node    { return []Node{e.Expr} }

type Goto struct {
	IsBreak bool
	IsCont  bool
	Pos     cpp.FilePos
	Label   string
}

func (g *Goto) GetPos() cpp.FilePos { return g.Pos }
func (*Goto) Children() []Node      { return []Node{} }

type LabeledStmt struct {
	Pos       cpp.FilePos
	AnonLabel string
	Label     string
	Stmt      Node
	IsCase    bool
	IsDefault bool
}

func (l *LabeledStmt) GetPos() cpp.FilePos { return l.Pos }
func (l *LabeledStmt) Children() []Node    { return []Node{l.Stmt} }

type If struct {
	Pos   cpp.FilePos
	Cond  Node
	Stmt  Node
	Else  Node
	LElse string
}

func (i *If) GetPos() cpp.FilePos { return i.Pos }
func (i *If) Children() []Node    { return []Node{i.Cond, i.Stmt, i.Else} }

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
func (sw *Switch) Children() []Node    { return []Node{sw.Expr, sw.Stmt} }

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
func (f *For) Children() []Node    { return []Node{f.Init, f.Cond, f.Step, f.Body} }

type While struct {
	Pos    cpp.FilePos
	Cond   Node
	Body   Node
	LStart string
	LEnd   string
}

func (w *While) GetPos() cpp.FilePos { return w.Pos }
func (w *While) Children() []Node    { return []Node{w.Cond, w.Body} }

type DoWhile struct {
	Pos    cpp.FilePos
	Cond   Node
	Body   Node
	LStart string
	LCond  string
	LEnd   string
}

func (d *DoWhile) GetPos() cpp.FilePos { return d.Pos }
func (d *DoWhile) Children() []Node    { return []Node{d.Body, d.Cond} }

type Unop struct {
	Op      cpp.TokenKind
	Pos     cpp.FilePos
	Operand Node
	Type    CType
}

func (u *Unop) GetType() CType      { return u.Type }
func (u *Unop) GetPos() cpp.FilePos { return u.Pos }
func (u *Unop) Children() []Node    { return []Node{u.Operand} }

type Binop struct {
	Op   cpp.TokenKind
	Pos  cpp.FilePos
	L    Node
	R    Node
	Type CType
}

func (b *Binop) GetType() CType      { return b.Type }
func (b *Binop) GetPos() cpp.FilePos { return b.Pos }
func (b *Binop) Children() []Node    { return []Node{b.L, b.R} }

type Function struct {
	Name         string
	Pos          cpp.FilePos
	FuncType     *FunctionType
	ParamSymbols []*LSymbol
	Body         []Node
}

func (f *Function) GetType() CType      { return f.FuncType }
func (f *Function) GetPos() cpp.FilePos { return f.Pos }
func (f *Function) Children() []Node    { return f.Body }

type Call struct {
	Pos      cpp.FilePos
	FuncLike Expr
	Args     []Expr
	Type     CType
}

func (c *Call) GetType() CType      { return c.Type }
func (c *Call) GetPos() cpp.FilePos { return c.Pos }
func (c *Call) Children() []Node {
	ret := []Node{c.FuncLike}
	for _, n := range c.Args {
		ret = append(ret, n)
	}
	return ret
}

type DeclList struct {
	Pos         cpp.FilePos
	Symbols     []Symbol
	Inits       []Node
	FoldedInits []*FoldedConstant
}

func (d *DeclList) GetPos() cpp.FilePos { return d.Pos }
func (d *DeclList) Children() []Node    { return d.Inits }

type String struct {
	Pos   cpp.FilePos
	Val   string
	Label string
}

func (s *String) GetType() CType      { return &Ptr{CChar} }
func (s *String) GetPos() cpp.FilePos { return s.Pos }
func (*String) Children() []Node      { return []Node{} }

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
func (*Ident) Children() []Node      { return []Node{} }
