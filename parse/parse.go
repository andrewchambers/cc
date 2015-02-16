package parse

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"os"
	"runtime/debug"
	"strconv"
)

// Storage class
type SClass int

const (
	SC_AUTO SClass = iota
	SC_REGISTER
	SC_STATIC
	SC_GLOBAL
)

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
	if os.Getenv("CCDEBUG") == "true" {
		err = fmt.Errorf("%s\n%s", err, debug.Stack())
	}
	err = cpp.ErrWithLoc(err, pos)
	panic(parseErrorBreakOut{err})
}

func (p *parser) error(m string, vals ...interface{}) {
	err := fmt.Errorf("syntax error: "+m, vals...)
	if os.Getenv("CCDEBUG") == "true" {
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

	for p.curt.Kind != cpp.EOF {
		p.parseDeclaration(true)
	}

}

func (p *parser) parseStatement() {

	if p.nextt.Kind == ':' {
		p.expect(cpp.IDENT)
		p.expect(':')
		p.parseStatement()
		return
	}

	switch p.curt.Kind {
	case cpp.GOTO:
		p.next()
		p.expect(cpp.IDENT)
		p.expect(';')
	case ';':
		p.next()
	case cpp.RETURN:
		p.next()
		p.parseExpression()
		p.expect(';')
	case cpp.WHILE:
		p.parseWhile()
	case cpp.DO:
		p.parseDoWhile()
	case cpp.FOR:
		p.parseFor()
	case cpp.IF:
		p.parseIf()
	case '{':
		p.parseBlock()
	default:
		p.parseExpression()
		p.expect(';')
	}
}

func (p *parser) parseIf() {

	p.expect(cpp.IF)
	p.expect('(')
	p.parseExpression()
	p.expect(')')
	p.parseStatement()
	if p.curt.Kind == cpp.ELSE {
		p.next()
		p.parseStatement()
	}
}

func (p *parser) parseFor() {

	p.expect(cpp.FOR)
	p.expect('(')
	if p.curt.Kind != ';' {
		p.parseExpression()
	}
	p.expect(';')
	if p.curt.Kind != ';' {
		p.parseExpression()
	}
	p.expect(';')
	if p.curt.Kind != ')' {
		p.parseExpression()
	}
	p.expect(')')
	p.parseStatement()
}

func (p *parser) parseWhile() {

	p.expect(cpp.WHILE)
	p.expect('(')
	p.parseExpression()
	p.expect(')')
	p.parseStatement()
}

func (p *parser) parseDoWhile() {

	p.expect(cpp.DO)
	p.parseStatement()
	p.expect(cpp.WHILE)
	p.expect('(')
	p.parseExpression()
	p.expect(')')
	p.expect(';')
}

func (p *parser) parseBlock() {

	p.expect('{')
	for p.curt.Kind != '}' {
		p.parseStatement()
	}
	p.expect('}')
}

func (p *parser) parseFuncBody() {

	for p.curt.Kind != '}' {
		p.parseStatement()
	}
}

func (p *parser) parseDeclaration(isGlobal bool) {

	firstDecl := true
	_, ty := p.parseDeclarationSpecifiers()
	for {
		_, _ = p.parseDeclarator(ty)

		if firstDecl && isGlobal {
			// if declaring a function
			if p.curt.Kind == '{' {
				p.expect('{')
				p.parseFuncBody()
				p.expect('}')
				return
			}
		}

		if p.curt.Kind == '=' {
			p.next()
			p.parseInitializer()
		}
		if p.curt.Kind != ',' {
			break
		}
		p.next()
		firstDecl = false
	}
	if p.curt.Kind != ';' {
		p.errorPos("expected '=', ',' or ';'", p.curt.Pos)
	}
	p.expect(';')
}

func (p *parser) parseParameterDeclaration() {

	_, ty := p.parseDeclarationSpecifiers()
	p.parseDeclarator(ty)
}

func (p *parser) parseDeclarationSpecifiers() (SClass, CType) {

	// These are assumed.
	sc := SC_AUTO
	ty := CInt
	for {
		switch p.curt.Kind {
		case cpp.REGISTER:
		case cpp.EXTERN:
		case cpp.STATIC:
		case cpp.TYPEDEF: // Typedef is actually a storage class like static.
		case cpp.VOID:
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
			p.parseStruct()
			return sc, ty
		case cpp.UNION:
		default:
			return sc, ty
		}
		p.next()
	}
}

// Declarator
// ----------
//
// A declarator is the part of a declaration that specifies
// the name that is to be introduced into the program.
//
// unsigned int a, *b, **c, *const*d *volatile*e ;
//              ^  ^^  ^^^  ^^^^^^^^ ^^^^^^^^^^^
//
// Direct Declarator
// -----------------
//
// A direct declarator is missing the pointer prefix.
//
// e.g.
// unsigned int *a[32], b[];
//               ^^^^^  ^^^
//
// Abstract Declarator
// -------------------
//
// A delcarator missing an identifier.

func (p *parser) parseDeclarator(basety CType) (string, CType) {

	for p.curt.Kind == cpp.CONST || p.curt.Kind == cpp.VOLATILE {
		p.next()
	}
	switch p.curt.Kind {
	case '*':
		p.next()
		name, ty := p.parseDeclarator(basety)
		return name, &Ptr{ty}
	case '(':
		p.next()
		name, ty := p.parseDeclarator(basety)
		p.expect(')')
		return name, p.parseDeclaratorTail(ty)
	case cpp.IDENT:
		name := p.curt.Val
		p.next()
		return name, p.parseDeclaratorTail(basety)
	default:
		p.errorPos(fmt.Sprintf("expected ident, '(' or '*' but got %s", p.curt.Kind), p.curt.Pos)
	}
	panic("unreachable")
}

func (p *parser) parseDeclaratorTail(basety CType) CType {

	ret := basety
	for {
		switch p.curt.Kind {
		case '[':
			p.next()
			if p.curt.Kind != ']' {
				p.parseAssignmentExpression()
			}
			p.expect(']')
			ret = &Array{MemberType: ret}
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
			return ret
		}
	}
}

func (p *parser) parseInitializer() {

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

func (p *parser) parseExpression() Node {
	var ret Node
	for {
		ret = p.parseAssignmentExpression()
		if p.curt.Kind != ',' {
			break
		}
		p.next()
	}
	return ret
}

func (p *parser) parseAssignmentExpression() Node {
	l := p.parseConditionalExpression()
	if isAssignmentOperator(p.curt.Kind) {
		p.next()
		p.parseAssignmentExpression()
	}
	return l
}

// Aka Ternary operator.
func (p *parser) parseConditionalExpression() Node {
	return p.parseLogicalOrExpression()
}

func (p *parser) parseLogicalOrExpression() Node {
	l := p.parseLogicalAndExpression()
	for p.curt.Kind == cpp.LOR {
		p.next()
		p.parseLogicalAndExpression()
	}
	return l
}

func (p *parser) parseLogicalAndExpression() Node {
	l := p.parseInclusiveOrExpression()
	for p.curt.Kind == cpp.LAND {
		p.next()
		p.parseInclusiveOrExpression()
	}
	return l
}

func (p *parser) parseInclusiveOrExpression() Node {
	l := p.parseExclusiveOrExpression()
	for p.curt.Kind == '|' {
		p.next()
		p.parseExclusiveOrExpression()
	}
	return l
}

func (p *parser) parseExclusiveOrExpression() Node {
	l := p.parseAndExpression()
	for p.curt.Kind == '^' {
		p.next()
		p.parseAndExpression()
	}
	return l
}

func (p *parser) parseAndExpression() Node {
	l := p.parseEqualityExpression()
	for p.curt.Kind == '&' {
		p.next()
		p.parseEqualityExpression()
	}
	return l
}

func (p *parser) parseEqualityExpression() Node {
	l := p.parseRelationalExpression()
	for p.curt.Kind == cpp.EQL || p.curt.Kind == cpp.NEQ {
		p.next()
		p.parseRelationalExpression()
	}
	return l
}

func (p *parser) parseRelationalExpression() Node {
	l := p.parseShiftExpression()
	for p.curt.Kind == '>' || p.curt.Kind == '<' || p.curt.Kind == cpp.LEQ || p.curt.Kind == cpp.GEQ {
		p.next()
		p.parseShiftExpression()
	}
	return l
}

func (p *parser) parseShiftExpression() Node {
	l := p.parseAdditiveExpression()
	for p.curt.Kind == cpp.SHL || p.curt.Kind == cpp.SHR {
		p.next()
		p.parseAdditiveExpression()
	}
	return l
}

func (p *parser) parseAdditiveExpression() Node {
	l := p.parseMultiplicativeExpression()
	for p.curt.Kind == '+' || p.curt.Kind == '-' {
		p.next()
		p.parseMultiplicativeExpression()
	}
	return l
}

func (p *parser) parseMultiplicativeExpression() Node {
	l := p.parseCastExpression()
	for p.curt.Kind == '*' || p.curt.Kind == '/' || p.curt.Kind == '%' {
		p.next()
		p.parseCastExpression()
	}
	return l
}

func (p *parser) parseCastExpression() Node {
	// Cast
	return p.parseUnaryExpression()
	panic("unreachable")
}

func (p *parser) parseUnaryExpression() Node {
	switch p.curt.Kind {
	case cpp.INC, cpp.DEC:
		p.next()
		p.parseUnaryExpression()
	case '*', '+', '-', '!', '~', '&':
		p.next()
		p.parseCastExpression()
	default:
		return p.parsePostfixExpression()
	}
	panic("unreachable")
}

func (p *parser) parsePostfixExpression() Node {
	l := p.parsePrimaryExpression()
loop:
	for {
		switch p.curt.Kind {
		case '[':
			p.next()
			p.parseExpression()
			p.expect(']')
		case '.', cpp.ARROW:
			p.next()
			// XXX is a typename valid here too?
			p.expect(cpp.IDENT)
		case '(':
			p.next()
			if p.curt.Kind != ')' {
				for {
					p.parseExpression()
					if p.curt.Kind == ',' {
						p.next()
						continue
					}
					break
				}
			}
			p.expect(')')
		case cpp.INC:
			p.next()
		case cpp.DEC:
			p.next()
		default:
			break loop
		}
	}
	return l
}

func constantToNode(t *cpp.Token) (Node, error) {
	switch t.Kind {
	case cpp.INT_CONSTANT:
		v, err := strconv.ParseInt(t.Val, 0, 64)
		return &Constant{
			Val:  v,
			Pos:  t.Pos,
			Type: CInt,
		}, err
	default:
		return nil, fmt.Errorf("internal error - %s", t.Kind)
	}
}

func (p *parser) parsePrimaryExpression() Node {
	switch p.curt.Kind {
	case cpp.IDENT:
		p.next()
	case cpp.INT_CONSTANT:
		t := p.curt
		p.next()
		n, err := constantToNode(t)
		if err != nil {
			p.errorPos(err.Error(), t.Pos)
		}
		return n
	case cpp.CHAR_CONSTANT:
		p.next()
	case cpp.STRING:
		p.next()
	case '(':
		p.next()
		p.parseExpression()
		p.expect(')')
	default:
		p.errorPos("expected an identifier, constant, string or expression", p.curt.Pos)
	}
	panic("unreachable")
}

func (p *parser) parseStruct() CType {
	p.expect(cpp.STRUCT)
	if p.curt.Kind == cpp.IDENT {
		p.next()
	}
	if p.curt.Kind == '{' {
		p.next()
		for {
			if p.curt.Kind == '}' {
				break
			}
			_, basety := p.parseDeclarationSpecifiers()
			for {
				p.parseDeclarator(basety)
				if p.curt.Kind == ',' {
					p.next()
					continue
				}
				break
			}
			p.expect(';')
		}
		p.expect('}')
	}
	return nil
}
