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
const ParseTrace bool = true

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
		p.parseDeclarator()
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

func (p *parser) parseParameterDeclaration() {
	trace()
	p.parseDeclarationSpecifiers()
	p.parseDeclarator()
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
	trace()
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
			break loop
		case cpp.IDENT:

		default:
			break loop
		}
		p.next()
	}
	switch p.curt.Kind {
	case '[':
		p.next()
		if p.curt.Kind != ']' {
			p.parseAssignmentExpression()
		}
		p.expect(']')
	case '(':
		p.next()
		if p.curt.Kind != ')' {
			for {
				p.parseParameterDeclaration()
				if p.curt.Kind == ',' {
					p.next()
					continue
				}
				break
			}
		}
		p.expect(')')
	default:
		return
	}
}

func (p *parser) parseInitializer() {
	trace()
	p.next()
}

func isAssignmentOperator(k cpp.TokenKind) bool {
	switch k {
	case '=', cpp.ADD_ASSIGN, cpp.SUB_ASSIGN, cpp.MUL_ASSIGN, cpp.QUO_ASSIGN, cpp.REM_ASSIGN,
		cpp.AND_ASSIGN, cpp.OR_ASSIGN, cpp.XOR_ASSIGN, cpp.SHL_ASSIGN, cpp.SHR_ASSIGN:
		return true
	}
	return false
}

func (p *parser) parseExpression() {
	trace()
	for {
		p.parseAssignmentExpression()
		if p.curt.Kind != ',' {
			break
		}
		p.next()
	}
}

func (p *parser) parseAssignmentExpression() {
	trace()
	p.parseConditionalExpression()
	if isAssignmentOperator(p.curt.Kind) {
		p.next()
		p.parseAssignmentExpression()
	}
}

// Aka Ternary operator.
func (p *parser) parseConditionalExpression() {
	trace()
	p.parseLogicalOrExpression()
}

func (p *parser) parseLogicalOrExpression() {
	trace()
	p.parseLogicalAndExpression()
	for p.curt.Kind == cpp.LOR {
		p.next()
		p.parseLogicalAndExpression()
	}
}

func (p *parser) parseLogicalAndExpression() {
	trace()
	p.parseInclusiveOrExpression()
	for p.curt.Kind == cpp.LAND {
		p.next()
		p.parseInclusiveOrExpression()
	}
}

func (p *parser) parseInclusiveOrExpression() {
	trace()
	p.parseExclusiveOrExpression()
	for p.curt.Kind == '|' {
		p.next()
		p.parseExclusiveOrExpression()
	}
}

func (p *parser) parseExclusiveOrExpression() {
	trace()
	p.parseAndExpression()
	for p.curt.Kind == '^' {
		p.next()
		p.parseAndExpression()
	}
}

func (p *parser) parseAndExpression() {
	trace()
	p.parseEqualityExpression()
	for p.curt.Kind == '&' {
		p.next()
		p.parseEqualityExpression()
	}
}

func (p *parser) parseEqualityExpression() {
	trace()
	p.parseRelationalExpression()
	for p.curt.Kind == cpp.EQL || p.curt.Kind == cpp.NEQ {
		p.next()
		p.parseRelationalExpression()
	}
}

func (p *parser) parseRelationalExpression() {
	trace()
	p.parseShiftExpression()
	for p.curt.Kind == '>' || p.curt.Kind == '<' || p.curt.Kind == cpp.LEQ || p.curt.Kind == cpp.GEQ {
		p.next()
		p.parseShiftExpression()
	}
}

func (p *parser) parseShiftExpression() {
	trace()
	p.parseAdditiveExpression()
	for p.curt.Kind == cpp.SHL || p.curt.Kind == cpp.SHR {
		p.next()
		p.parseAdditiveExpression()
	}
}

func (p *parser) parseAdditiveExpression() {
	trace()
	p.parseMultiplicativeExpression()
	for p.curt.Kind == '+' || p.curt.Kind == '-' {
		p.next()
		p.parseMultiplicativeExpression()
	}
}

func (p *parser) parseMultiplicativeExpression() {
	trace()
	p.parseCastExpression()
	for p.curt.Kind == '*' || p.curt.Kind == '/' || p.curt.Kind == '%' {
		p.next()
		p.parseCastExpression()
	}
}

func (p *parser) parseCastExpression() {
	trace()
	// Cast
	p.parseUnaryExpression()
}

func (p *parser) parseUnaryExpression() {
	trace()
	switch p.curt.Kind {
	case cpp.INC, cpp.DEC:
		p.parseUnaryExpression()
	case '*', '+', '-', '!', '~', '&':
		p.parseCastExpression()
	default:
		p.parsePostfixExpression()
	}
}

func (p *parser) parsePostfixExpression() {
	p.next()
}
