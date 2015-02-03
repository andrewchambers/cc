package parse

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"os"
	"runtime"
	"runtime/debug"
)

// Useful for debugging syntax errors.
// Enabling this will cause parsing information to be printed to stderr.
// Also, more information will be given for parse errors.
var ParseTrace bool = false

func trace() {
	if !ParseTrace {
		return
	}
	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		return
	}
	fmt.Fprintf(os.Stderr, "%s:%d\n", runtime.FuncForPC(pc).Name(), line)
}

type parser struct {
	types       *scope
	decls       *scope
	tokChan     <-chan *cpp.Token
	curt, nextt *cpp.Token
}

type parseErrorBreakOut struct {
	err error
}

func Parse(<-chan *cpp.Token) (ret []Node, errRet error) {
	p := &parser{}
	p.types = newScope(nil)
	p.decls = newScope(nil)

	defer func() {
		if e := recover(); e != nil {
			peb := e.(parseErrorBreakOut) // Will re-panic if not a breakout.
			errRet = peb.err
		}
	}()
	p.next()
	p.next()
loop:
	for {
		trace()

		break loop
	}
	return ret, nil

}

func (p *parser) error(m string, vals ...interface{}) {
	err := fmt.Errorf("syntax error: "+m, vals...)
	if ParseTrace {
		err = fmt.Errorf("%s\n%s", err, debug.Stack())
	}
	panic(parseErrorBreakOut{err})
}

func (p *parser) expect(k cpp.TokenKind) {
	if p.curt.Kind != k {
		p.error("expected %s got %s at %s", k, p.curt.Val, p.curt.Pos)
	}
	p.next()
}

func (p *parser) next() {
	p.curt = p.nextt
	// XXX error handling from previous stages?
	t := <-p.tokChan
	p.nextt = t
}

func (*parser) parseTranslationUnit() {

}
