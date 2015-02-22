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

	lcounter int
}

type parseErrorBreakOut struct {
	err error
}

func (p *parser) NextLabel() string {
	p.lcounter += 1
	return fmt.Sprintf(".L%d", p.lcounter)
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

func (p *parser) tryCastToBool(n Expr) *Cast {
	if IsIntType(n.GetType()) || IsPtrType(n.GetType()) {
		return &Cast{
			Pos:     n.GetPos(),
			Operand: n,
			Type:    CBool,
		}
	}
	p.errorPos(n.GetPos(), "bad cast")
	panic("unreachable")

}

func (p *parser) parseTranslationUnit() []Node {
	var topLevels []Node
	for p.curt.Kind != cpp.EOF {
		toplevel := p.parseDecl(true)
		topLevels = append(topLevels, toplevel)
	}
	return topLevels
}

func isDeclStart(t cpp.TokenKind) bool {
	switch t {
	case cpp.STATIC, cpp.VOLATILE, cpp.STRUCT, cpp.CHAR, cpp.INT, cpp.SHORT, cpp.LONG,
		cpp.UNSIGNED, cpp.SIGNED, cpp.FLOAT, cpp.DOUBLE:
		return true
	}
	return false
}

func (p *parser) parseStmt() Node {
	if p.nextt.Kind == ':' {
		p.expect(cpp.IDENT)
		p.expect(':')
		return p.parseStmt()
	}
	if isDeclStart(p.curt.Kind) {
		return p.parseDecl(false)
	} else {
		switch p.curt.Kind {
		case cpp.GOTO:
			p.next()
			p.expect(cpp.IDENT)
			p.expect(';')
		case ';':
			pos := p.curt.Pos
			p.next()
			return &EmptyStmt{
				Pos: pos,
			}
		case cpp.RETURN:
			return p.parseReturn()
		case cpp.WHILE:
			return p.parseWhile()
		case cpp.DO:
			p.parseDoWhile()
		case cpp.FOR:
			return p.parseFor()
		case cpp.IF:
			return p.parseIf()
		case '{':
			return p.parseBlock()
		default:
			pos := p.curt.Pos
			expr := p.parseExpr()
			p.expect(';')
			return &ExprStmt{
				Pos:  pos,
				Expr: expr,
			}
		}
	}
	panic("unreachable.")
}

func (p *parser) parseReturn() Node {
	pos := p.curt.Pos
	p.expect(cpp.RETURN)
	expr := p.parseExpr()
	p.expect(';')
	return &Return{
		Pos:  pos,
		Expr: expr,
	}
}

func (p *parser) parseIf() Node {
	ifpos := p.curt.Pos
	lelse := p.NextLabel()
	p.expect(cpp.IF)
	p.expect('(')
	expr := p.parseExpr()
	expr = p.tryCastToBool(expr)
	p.expect(')')
	stmt := p.parseStmt()
	var els Node
	if p.curt.Kind == cpp.ELSE {
		p.next()
		els = p.parseStmt()
	}
	return &If{
		Pos:   ifpos,
		Cond:  expr,
		Stmt:  stmt,
		Else:  els,
		LElse: lelse,
	}
}

func (p *parser) parseFor() Node {
	pos := p.curt.Pos
	lstart := p.NextLabel()
	lend := p.NextLabel()
	var init, cond, step Expr
	p.expect(cpp.FOR)
	p.expect('(')
	if p.curt.Kind != ';' {
		init = p.parseExpr()
	}
	p.expect(';')
	if p.curt.Kind != ';' {
		cond = p.parseExpr()
	}
	p.expect(';')
	if p.curt.Kind != ')' {
		step = p.parseExpr()
	}
	p.expect(')')
	body := p.parseStmt()
	return &For{
		Pos:    pos,
		Init:   init,
		Cond:   cond,
		Step:   step,
		Body:   body,
		LStart: lstart,
		LEnd:   lend,
	}
}

func (p *parser) parseWhile() Node {
	pos := p.curt.Pos
	lstart := p.NextLabel()
	lend := p.NextLabel()
	p.expect(cpp.WHILE)
	p.expect('(')
	expr := p.tryCastToBool(p.parseExpr())
	p.expect(')')
	body := p.parseStmt()
	return &While{
		Pos:    pos,
		Cond:   expr,
		Body:   body,
		LStart: lstart,
		LEnd:   lend,
	}
}

func (p *parser) parseDoWhile() {
	p.expect(cpp.DO)
	p.parseStmt()
	p.expect(cpp.WHILE)
	p.expect('(')
	p.parseExpr()
	p.expect(')')
	p.expect(';')
}

func (p *parser) parseBlock() *CompndStmt {
	var stmts []Node
	pos := p.curt.Pos
	p.expect('{')
	for p.curt.Kind != '}' {
		stmts = append(stmts, p.parseStmt())
	}
	p.expect('}')
	return &CompndStmt{
		Pos:  pos,
		Body: stmts,
	}
}

func (p *parser) parseFuncBody(f *Function) {
	for p.curt.Kind != '}' {
		stmt := p.parseStmt()
		f.Body = append(f.Body, stmt)
	}
}

func (p *parser) parseDecl(isGlobal bool) Node {
	firstDecl := true
	declPos := p.curt.Pos
	var name *cpp.Token
	declList := &DeclList{}
	_, ty := p.parseDeclSpecifiers()
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
		var sym Symbol
		if isGlobal {
			sym = &GSymbol{
				Label: name.Val,
				Type:  ty,
			}
		} else {
			sym = &LSymbol{
				Type: ty,
			}
		}
		err := p.decls.define(name.Val, sym)
		if err != nil {
			p.errorPos(name.Pos, err.Error())
		}
		declList.Symbols = append(declList.Symbols, sym)
		var init Node
		var initPos cpp.FilePos
		var folded *FoldedConstant
		if p.curt.Kind == '=' {
			p.next()
			initPos = p.curt.Pos
			init = p.parseInitializer()
			folded, err = Fold(init)
			if err != nil {
				folded = nil
				if isGlobal {
					p.errorPos(initPos, err.Error())
				}
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

func (p *parser) parseParamDecl() (*cpp.Token, CType) {
	_, ty := p.parseDeclSpecifiers()
	return p.parseDeclarator(ty)
}

func (p *parser) parseDeclSpecifiers() (SClass, CType) {
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
// A declarator is the part of a Decl that specifies
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
			var dimn Node
			if p.curt.Kind != ']' {
				dimn = p.parseAssignmentExpr()
			}
			p.expect(']')
			dim, err := Fold(dimn)
			if err != nil {
				p.errorPos(dimn.GetPos(), "invalid constant Expr for array dimensions")
			}
			ret = &Array{
				Dim:        int(dim.Val),
				MemberType: ret,
			}
		case '(':
			fret := &FunctionType{}
			fret.RetType = basety
			p.next()
			if p.curt.Kind != ')' {
				for {
					pnametok, pty := p.parseParamDecl()
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
	return p.parseAssignmentExpr()
}

func isAssignmentOperator(k cpp.TokenKind) bool {
	switch k {
	case '=', cpp.ADD_ASSIGN, cpp.SUB_ASSIGN, cpp.MUL_ASSIGN, cpp.QUO_ASSIGN, cpp.REM_ASSIGN,
		cpp.AND_ASSIGN, cpp.OR_ASSIGN, cpp.XOR_ASSIGN, cpp.SHL_ASSIGN, cpp.SHR_ASSIGN:
		return true
	}
	return false
}

func (p *parser) parseExpr() Expr {
	var ret Expr
	for {
		ret = p.parseAssignmentExpr()
		if p.curt.Kind != ',' {
			break
		}
		p.next()
	}
	return ret
}

func (p *parser) parseAssignmentExpr() Expr {
	l := p.parseCondExpr()
	if isAssignmentOperator(p.curt.Kind) {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseAssignmentExpr()
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
func (p *parser) parseCondExpr() Expr {
	return p.parseLogOrExpr()
}

func (p *parser) parseLogOrExpr() Expr {
	l := p.parseLogAndExpr()
	for p.curt.Kind == cpp.LOR {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseLogAndExpr()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseLogAndExpr() Expr {
	l := p.parseInclusiveOrExpr()
	for p.curt.Kind == cpp.LAND {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseInclusiveOrExpr()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseInclusiveOrExpr() Expr {
	l := p.parseExclusiveOrExpr()
	for p.curt.Kind == '|' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseExclusiveOrExpr()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseExclusiveOrExpr() Expr {
	l := p.parseAndExpr()
	for p.curt.Kind == '^' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseAndExpr()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseAndExpr() Expr {
	l := p.parseEqualityExpr()
	for p.curt.Kind == '&' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseEqualityExpr()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseEqualityExpr() Expr {
	l := p.parseRelationalExpr()
	for p.curt.Kind == cpp.EQL || p.curt.Kind == cpp.NEQ {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseRelationalExpr()
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

func (p *parser) parseRelationalExpr() Expr {
	l := p.parseShiftExpr()
	for p.curt.Kind == '>' || p.curt.Kind == '<' || p.curt.Kind == cpp.LEQ || p.curt.Kind == cpp.GEQ {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseShiftExpr()
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

func (p *parser) parseShiftExpr() Expr {
	l := p.parseAdditiveExpr()
	for p.curt.Kind == cpp.SHL || p.curt.Kind == cpp.SHR {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseAdditiveExpr()
		l = &Binop{
			Pos: pos,
			Op:  op,
			L:   l,
			R:   r,
		}
	}
	return l
}

func (p *parser) parseAdditiveExpr() Expr {
	l := p.parseMultiplicativeExpr()
	for p.curt.Kind == '+' || p.curt.Kind == '-' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseMultiplicativeExpr()
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

func (p *parser) parseMultiplicativeExpr() Expr {
	l := p.parseCastExpr()
	for p.curt.Kind == '*' || p.curt.Kind == '/' || p.curt.Kind == '%' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.parseCastExpr()
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

func (p *parser) parseCastExpr() Expr {
	// Cast
	return p.parseUnaryExpr()
}

func (p *parser) parseUnaryExpr() Expr {
	switch p.curt.Kind {
	case cpp.INC, cpp.DEC:
		p.next()
		p.parseUnaryExpr()
	case '*', '+', '-', '!', '~', '&':
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		operand := p.parseCastExpr()
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
		return p.parsePostfixExpr()
	}
	panic("unreachable")
}

func (p *parser) parsePostfixExpr() Expr {
	l := p.parsePrimaryExpr()
loop:
	for {
		switch p.curt.Kind {
		case '[':
			_, isArr := l.GetType().(*Array)
			_, isPtr := l.GetType().(*Ptr)
			if !isArr && !isPtr {
				p.errorPos(p.curt.Pos, "Can only index into array or pointer types")
			}
			p.next()
			idx := p.parseExpr()
			p.expect(']')
			l = &Index{
				Arr: l,
				Idx: idx,
			}
		case '.', cpp.ARROW:
			p.next()
			// XXX is a typename valid here too?
			p.expect(cpp.IDENT)
		case '(':
			p.next()
			if p.curt.Kind != ')' {
				for {
					p.parseExpr()
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

func constantToExpr(t *cpp.Token) (Expr, error) {
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

func (p *parser) parsePrimaryExpr() Expr {
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
		n, err := constantToExpr(t)
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
		p.parseExpr()
		p.expect(')')
	default:
		p.errorPos(p.curt.Pos, "expected an identifier, constant, string or Expr")
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
			_, basety := p.parseDeclSpecifiers()
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
