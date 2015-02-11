package parse

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"os"
	"runtime"
	"runtime/debug"
)

// Storage class
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
	pp          *cpp.Preprocessor
	curt, nextt *cpp.Token
}

type parseErrorBreakOut struct {
	err error
}

func Parse(pp *cpp.Preprocessor) (errRet error) {
	p := &parser{}
	p.pp = pp
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

func (p *parser) errorPos(m string, pos cpp.FilePos, vals ...interface{}) {
	err := fmt.Errorf("syntax error: "+m, vals...)
	if ParseTrace {
		err = fmt.Errorf("%s\n%s", err, debug.Stack())
	}
	err = cpp.ErrWithLoc(err, pos)
	panic(parseErrorBreakOut{err})
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
		p.errorPos("expected %s got %s", p.curt.Pos, k, p.curt.Kind)
	}
	p.next()
}

func (p *parser) next() {
	p.curt = p.nextt
	t, err := p.pp.Next()
	if err != nil {
		p.error(err.Error())
	}
	p.nextt = t
}

func (p *parser) parseTranslationUnit() {
	trace()
	for p.curt.Kind != cpp.EOF {
		p.parseDeclaration()
	}
	trace()
}

func (p *parser) parseDeclaration() {
	trace()
	p.parseDeclarationSpecifiers()
	for {
		p.parseDeclarator(false)
		if p.curt.Kind == '=' {
			p.next()
			p.parseInitializer()
		}
		if p.curt.Kind != ',' {
			break
		}
	}
	if p.curt.Kind != ';' {
		p.errorPos("expected '=', ',' or ';'", p.curt.Pos)
	}
	p.expect(';')
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

func (p *parser) parseDeclarator(abstract bool) {
	trace()
loop:
	for {
		switch p.curt.Kind {
		case '*':
		case cpp.CONST:
		case cpp.VOLATILE:
		case '(':
			p.next()
			p.parseDeclarator(abstract)
			p.expect(')')
			break loop
		case cpp.IDENT:
			if abstract {
				break loop
			}
		default:
			break loop
		}
		p.next()
	}
	switch p.curt.Kind {
	case '[':
		p.expect(']')
	case '(':
		switch p.curt.Kind {
		case cpp.IDENT:
		case ')':
			break
		}
		p.expect(')')
	default:
		return
	}
}

func (p *parser) parseInitializer() {
	p.next()
}
