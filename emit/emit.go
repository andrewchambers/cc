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
	e.emiti("mov $0, %%rax\n")
	e.emiti("ret\n")
}
