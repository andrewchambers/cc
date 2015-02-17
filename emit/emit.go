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
	default:
		panic(e)
	}
}
