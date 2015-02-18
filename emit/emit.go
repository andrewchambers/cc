package emit

import (
	"fmt"
	"github.com/andrewchambers/cc/parse"
	"io"
)

type emitter struct {
	o io.Writer
}

func Emit(toplevels []parse.Node, o io.Writer) error {
	e := &emitter{
		o: o,
	}
	for _, tl := range toplevels {
		switch tl := tl.(type) {
		case *parse.Function:
			e.emitFunction(tl)
		case *parse.DeclList:
			for _, decl := range tl.Symbols {
				global, ok := decl.(*parse.GSymbol)
				if !ok {
					panic("internal error")
				}
				e.emitGlobal(global)
			}
		default:
			panic(tl)
		}
	}
	return nil
}

func (e *emitter) emit(s string, args ...interface{}) {
	fmt.Fprintf(e.o, s, args...)
}

func (e *emitter) emiti(s string, args ...interface{}) {
	e.emit("  "+s, args...)
}

func isPtr(ty parse.CType) bool {
	_, ok := ty.(*parse.Ptr)
	return ok
}

func (e *emitter) emitGlobal(g *parse.GSymbol) {
	e.emit(".global %s\n", g.Label)
	e.emit("%s:\n", g.Label)
	if g.Init != nil {
		return
	}
	switch {
	case g.Type == parse.CInt:
		e.emit(".dword 0\n")
	case isPtr(g.Type):
		e.emit(".qword 0\n")
	default:
	}
}

func (e *emitter) emitFunction(f *parse.Function) {
	e.emit(".global %s\n", f.Name)
	e.emit("%s:\n", f.Name)
	e.emiti("pushq %%rbp\n")
	e.emiti("movq %%rsp, %%rbp\n")
	for _, stmt := range f.Body {
		e.emitStatement(f, stmt)
	}
	e.emiti("leave\n")
	e.emiti("ret\n")
}

func (e *emitter) emitStatement(f *parse.Function, stmt parse.Node) {
	switch stmt := stmt.(type) {
	case *parse.Return:
		e.emitReturn(f, stmt)
	default:
		panic(stmt)
	}
}

func (e *emitter) emitReturn(f *parse.Function, r *parse.Return) {
	e.emitExpr(f, r.Expr)
	e.emiti("leave\n")
	e.emiti("ret\n")
}

func (e *emitter) emitExpr(f *parse.Function, expr parse.Node) {
	switch expr := expr.(type) {
	case *parse.Constant:
		e.emiti("movq $%v, %%rax\n", expr.Val)
	case *parse.Binop:
		e.emitBinop(f, expr)
	default:
		panic(e)
	}
}

func (e *emitter) emitBinop(f *parse.Function, b *parse.Binop) {
	e.emitExpr(f, b.L)
	e.emiti("pushq %%rax\n")
	e.emitExpr(f, b.R)
	e.emiti("popq %%rbx\n")
	switch {
	case b.Type == parse.CInt:
		switch b.Op {
		case '+':
			e.emiti("addq %%rax, %%rbx\n")
		case '-':
			e.emiti("subq %%rax, %%rbx\n")
		case '*':
			e.emiti("imul %%rax, %%rbx\n")
		default:
			panic("unimplemented")
		}
		e.emiti("movq %%rbx, %%rax\n")
	default:
		panic(b.Type)
	}
}
