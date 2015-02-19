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

func Parse(pp *cpp.Preprocessor) (toplevels []Node, errRet error) {
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
	toplevels = p.parseTranslationUnit()
	return toplevels, nil
}

func (p *parser) errorPos(pos cpp.FilePos, m string, vals ...interface{}) {
	err := fmt.Errorf(m, vals...)
	if os.Getenv("CCDEBUG") == "true" {
		err = fmt.Errorf("%s\n%s", err, debug.Stack())
	}
	err = cpp.ErrWithLoc(err, pos)
	panic(parseErrorBreakOut{err})
}

func (p *parser) error(m string, vals ...interface{}) {
	err := fmt.Errorf(m, vals...)
	if os.Getenv("CCDEBUG") == "true" {
		err = fmt.Errorf("%s\n%s", err, debug.Stack())
	}
	panic(parseErrorBreakOut{err})
}

func (p *parser) expect(k cpp.TokenKind) {
	if p.curt.Kind != k {
		p.errorPos(p.curt.Pos, "expected %s got %s", k, p.curt.Kind)
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

func (p *parser) parseTranslationUnit() []Node {
	var topLevels []Node
	for p.curt.Kind != cpp.EOF {
		toplevel := p.parseDeclaration(true)
		topLevels = append(topLevels, toplevel)
	}
	return topLevels
}

func (p *parser) parseStatement() Node {
	if p.nextt.Kind == ':' {
		p.expect(cpp.IDENT)
		p.expect(':')
		return p.parseStatement()
	}
	switch p.curt.Kind {
	case cpp.GOTO:
		p.next()
		p.expect(cpp.IDENT)
		p.expect(';')
	case ';':
		p.next()
	case cpp.RETURN:
		return p.parseReturn()
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
		expr := p.parseExpression()
		p.expect(';')
		return expr
	}
	panic("unreachable.")
}

func (p *parser) parseReturn() Node {
	pos := p.curt.Pos
	p.expect(cpp.RETURN)
	expr := p.parseExpression()
	p.expect(';')
	return &Return{
		Pos:  pos,
		Expr: expr,
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

func (p *parser) parseFuncBody(f *Function) {
	for p.curt.Kind != '}' {
		stmt := p.parseStatement()
		f.Body = append(f.Body, stmt)
	}
}

func (p *parser) parseDeclaration(isGlobal bool) Node {
	firstDecl := true
	declPos := p.curt.Pos
	var name *cpp.Token
	declList := &DeclList{}
	_, ty := p.parseDeclarationSpecifiers()
	for {
		name, ty = p.parseDeclarator(ty)
		if name == nil {
			p.errorPos(declPos, "declarator requires a name")
		}
		if firstDecl && isGlobal {
			// if declaring a function
			if p.curt.Kind == '{' {
				fty, ok := ty.(*FunctionType)
				if !ok {
					p.errorPos(name.Pos, "expected a function")
				}
				f := &Function{
					Name:     name.Val,
					FuncType: fty,
					Pos:      declPos,
				}
				p.expect('{')
				p.parseFuncBody(f)
				p.expect('}')
				return f
			}
		}
		sym := &GSymbol{
			Label: name.Val,
			Type:  ty,
		}
		err := p.decls.define(name.Val, sym)
		if err != nil {
			p.errorPos(name.Pos, err.Error())
		}
		declList.Symbols = append(declList.Symbols, sym)
		var init Node
		var initPos cpp.FilePos
		if p.curt.Kind == '=' {
			p.next()
			initPos = p.curt.Pos
			init = p.parseInitializer()
		}
		folded, err := Fold(init)
		if err != nil {
			folded = nil
			if isGlobal {
				p.errorPos(initPos, err.Error())
			}
		}
		declList.Inits = append(declList.Inits, init)
		declList.FoldedInits = append(declList.FoldedInits, folded)
		if p.curt.Kind != ',' {
			break
		}
		p.next()
		firstDecl = false
	}
	if p.curt.Kind != ';' {
		p.errorPos(p.curt.Pos, "expected '=', ',' or ';'")
	}
	p.expect(';')
	return declList
}

func (p *parser) parseParameterDeclaration() (*cpp.Token, CType) {
	_, ty := p.parseDeclarationSpecifiers()
	return p.parseDeclarator(ty)
}

func (p *parser) parseDeclarationSpecifiers() (SClass, CType) {
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

func (p *parser) parseDeclarator(basety CType) (*cpp.Token, CType) {
	for p.curt.Kind == cpp.CONST || p.curt.Kind == cpp.VOLATILE {
		p.next()
	}
	switch p.curt.Kind {
	case '*':
		p.next()
		name, ty := p.parseDeclarator(basety)
		return name, &Ptr{ty}
	case '(':
		forward := &ForwardedType{}
		p.next()
		name, ty := p.parseDeclarator(forward)
		p.expect(')')
		forward.Type = p.parseDeclaratorTail(basety)
		return name, ty
	case cpp.IDENT:
		name := p.curt
		p.next()
		return name, p.parseDeclaratorTail(basety)
	default:
		p.errorPos(p.curt.Pos, "expected ident, '(' or '*' but got %s", p.curt.Kind)
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
			fret := &FunctionType{}
			fret.RetType = basety
			p.next()
			if p.curt.Kind != ')' {
				for {
					pnametok, pty := p.parseParameterDeclaration()
					pname := ""
					if pnametok != nil {
						pname = pnametok.Val
					}
					fret.ArgTypes = append(fret.ArgTypes, pty)
					fret.ArgNames = append(fret.ArgNames, pname)
					if p.curt.Kind == ',' {
						p.next()
						continue
					}
					break
				}
			}
			p.expect(')')
			ret = fret
		default:
			return ret
		}
	}
}

func (p *parser) parseInitializer() Node {
	return p.parseAssignmentExpression()
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
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseAssignmentExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
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
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseLogicalAndExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseLogicalAndExpression() Node {
	l := p.parseInclusiveOrExpression()
	for p.curt.Kind == cpp.LAND {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseInclusiveOrExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseInclusiveOrExpression() Node {
	l := p.parseExclusiveOrExpression()
	for p.curt.Kind == '|' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseExclusiveOrExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseExclusiveOrExpression() Node {
	l := p.parseAndExpression()
	for p.curt.Kind == '^' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseAndExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseAndExpression() Node {
	l := p.parseEqualityExpression()
	for p.curt.Kind == '&' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseEqualityExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseEqualityExpression() Node {
	l := p.parseRelationalExpression()
	for p.curt.Kind == cpp.EQL || p.curt.Kind == cpp.NEQ {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseRelationalExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseRelationalExpression() Node {
	l := p.parseShiftExpression()
	for p.curt.Kind == '>' || p.curt.Kind == '<' || p.curt.Kind == cpp.LEQ || p.curt.Kind == cpp.GEQ {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseShiftExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseShiftExpression() Node {
	l := p.parseAdditiveExpression()
	for p.curt.Kind == cpp.SHL || p.curt.Kind == cpp.SHR {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseAdditiveExpression()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseAdditiveExpression() Node {
	l := p.parseMultiplicativeExpression()
	for p.curt.Kind == '+' || p.curt.Kind == '-' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseMultiplicativeExpression()
		l = &Binop{
			Pos:  pos,
			Op:   op,
			L:    l,
			R:    r,
			Type: CInt,
		}
	}
	return l
}

func (p *parser) parseMultiplicativeExpression() Node {
	l := p.parseCastExpression()
	for p.curt.Kind == '*' || p.curt.Kind == '/' || p.curt.Kind == '%' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseCastExpression()
		l = &Binop{
			Pos:  pos,
			Op:   op,
			L:    l,
			R:    r,
			Type: CInt,
		}
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
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		operand := p.parseCastExpression()
		ty := operand.GetType()
		if op == '&' {
			ty = &Ptr{
				PointsTo: ty,
			}
		} else if op == '*' {
			ptr, ok := ty.(*Ptr)
			if !ok {
				p.errorPos(pos, "dereferencing requires a pointer type")
			}
			ty = ptr.PointsTo
		}
		return &Unop{
			Pos:     pos,
			Op:      op,
			Operand: operand,
			Type:    ty,
		}
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
		sym, err := p.decls.lookup(p.curt.Val)
		if err != nil {
			p.errorPos(p.curt.Pos, "undefined symbol %s", p.curt.Val)
		}
		p.next()
		return &Ident{
			Sym: sym,
		}
	case cpp.INT_CONSTANT:
		t := p.curt
		p.next()
		n, err := constantToNode(t)
		if err != nil {
			p.errorPos(t.Pos, err.Error())
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
		p.errorPos(p.curt.Pos, "expected an identifier, constant, string or expression")
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
