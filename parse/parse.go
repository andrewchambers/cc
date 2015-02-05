package parse

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"os"
	"runtime"
	"runtime/debug"
)

// Storage Class
type SClass int

const (
	SC_AUTO SClass = iota
	SC_REGISTER
	SC_STATIC
	SC_GLOBAL
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

func Parse(<-chan *cpp.Token) (errRet error) {
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
	p.parseTranslationUnit()
	return nil
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
	t := <-p.tokChan
	if t == nil {
		t = &cpp.Token{}
		t.Kind = cpp.EOF
	}
	if t.Kind == cpp.ERROR {
		p.error(t.Val)
	}
	p.nextt = t
}

func (p *parser) parseTranslationUnit() {
	trace()
	for p.curt.Kind != cpp.EOF {
		p.parseDeclaration()
	}
}

func (p *parser) parseDeclaration() {
	trace()
	p.parseDeclarationSpecifiers()
	for {
		p.parseDeclarator()
		if p.curt.Kind == '=' {
			p.next()
			p.parseInitializer()
		}
		if p.curt.Kind != ',' {
			break
		}
	}
}

func (p *parser) parseDeclarationSpecifiers() (SClass, CType) {
	trace()
	// These are assumed.
	sc := SC_AUTO
	ty := CInt
	for {
		switch p.curt.Kind {
		case cpp.REGISTER:
		case cpp.EXTERN:
		case cpp.STATIC:
		case cpp.TYPEDEF:
		case cpp.CHAR:
		case cpp.SHORT:
		case cpp.INT:
		case cpp.LONG:
		case cpp.FLOAT:
		case cpp.DOUBLE:
		case cpp.SIGNED:
		case cpp.UNSIGNED:
		case cpp.TYPENAME:
		case cpp.STRUCT:
		case cpp.UNION:
		default:
			return sc, ty
		}
		p.next()
	}
	panic("unreachable")
}

func (p *parser) parseDeclarator() {
loop:
	for {
		switch p.curt.Kind {
		case '*':
		case cpp.CONST:
		case cpp.VOLATILE:
		case '(':
			p.next()
			p.parseDeclarator()
			p.expect(')')
		case cpp.IDENT:
		default:
			break loop
		}
		p.next()
	}
	switch p.curt.Kind {
	case '[':
		p.expect(']')
	case '(':
		p.expect(')')
	default:
		return
	}
}

func (p *parser) parseInitializer() {
	p.next()
}
