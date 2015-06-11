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
	SC_TYPEDEF
	SC_GLOBAL
)

type parseErrorBreakOut struct {
	err error
}

type gotoFixup struct {
	actualLabel string
	g           *Goto
}

type parser struct {
	szdesc TargetSizeDesc

	types   *scope
	structs *scope
	decls   *scope

	tu *TranslationUnit

	pp          *cpp.Preprocessor
	curt, nextt *cpp.Token
	lcounter    int

	breakCounter int
	breaks       [2048]string
	contCounter  int
	continues    [2048]string

	switchCounter int
	switchs       [2048]*Switch

	// Map of goto labels to anonymous labels.
	labels map[string]string
	// All gotos found in the current function.
	// Needed so we can fix up forward references.
	gotos []gotoFixup
}

func (p *parser) pushScope() {
	p.decls = newScope(p.decls)
	p.structs = newScope(p.structs)
	p.types = newScope(p.types)
}

func (p *parser) popScope() {
	p.decls = p.decls.parent
	p.structs = p.structs.parent
	p.types = p.types.parent
}

func (p *parser) pushSwitch(s *Switch) {
	p.switchs[p.switchCounter] = s
	p.switchCounter += 1
}

func (p *parser) popSwitch() {
	p.switchCounter -= 1
}

func (p *parser) getSwitch() *Switch {
	if p.switchCounter == 0 {
		return nil
	}
	return p.switchs[p.switchCounter-1]
}

func (p *parser) pushBreak(blabel string) {
	p.breaks[p.breakCounter] = blabel
	p.breakCounter += 1
}

func (p *parser) pushCont(clabel string) {
	p.continues[p.contCounter] = clabel
	p.contCounter += 1
}

func (p *parser) popBreak() {
	p.breakCounter -= 1
	if p.breakCounter < 0 {
		panic("internal error")
	}
}

func (p *parser) popCont() {
	p.contCounter -= 1
	if p.contCounter < 0 {
		panic("internal error")
	}
}

func (p *parser) pushBreakCont(blabel, clabel string) {
	p.pushBreak(blabel)
	p.pushCont(clabel)
}

func (p *parser) popBreakCont() {
	p.popBreak()
	p.popCont()
}

func (p *parser) getBreakLabel() string {
	if p.breakCounter == 0 {
		return ""
	}
	return p.breaks[p.breakCounter-1]
}

func (p *parser) getContLabel() string {
	if p.contCounter == 0 {
		return ""
	}
	return p.continues[p.contCounter-1]
}

func (p *parser) nextLabel() string {
	p.lcounter += 1
	return fmt.Sprintf(".L%d", p.lcounter)
}

func (p *parser) addAnonymousString(s *String) {
	p.tu.AnonymousInits = append(p.tu.AnonymousInits, s)
}

func Parse(szdesc TargetSizeDesc, pp *cpp.Preprocessor) (tu *TranslationUnit, errRet error) {
	p := &parser{}
	p.szdesc = szdesc
	p.pp = pp
	p.types = newScope(nil)
	p.decls = newScope(nil)
	p.structs = newScope(nil)
	p.tu = &TranslationUnit{}

	defer func() {
		if e := recover(); e != nil {
			peb := e.(parseErrorBreakOut) // Will re-panic if not a breakout.
			errRet = peb.err
		}
	}()
	p.next()
	p.next()
	p.TUnit()
	return p.tu, nil
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

func (p *parser) ensureScalar(n Expr) {
	if !IsScalarType(n.GetType()) {
		p.errorPos(n.GetPos(), "expected scalar type")
	}
}

func (p *parser) TUnit() {
	for p.curt.Kind != cpp.EOF {
		toplevel := p.Decl(true)
		p.tu.TopLevels = append(p.tu.TopLevels, toplevel)
	}
}

func (p *parser) isDeclStart(t *cpp.Token) bool {
	switch t.Kind {
	case cpp.IDENT:
		_, err := p.decls.lookup(t.Val)
		if err != nil {
			return true
		}
	case cpp.STATIC, cpp.VOLATILE, cpp.STRUCT, cpp.CHAR, cpp.INT, cpp.SHORT, cpp.LONG,
		cpp.UNSIGNED, cpp.SIGNED, cpp.FLOAT, cpp.DOUBLE:
		return true
	}
	return false
}

func (p *parser) Stmt() Node {
	if p.nextt.Kind == ':' && p.curt.Kind == cpp.IDENT {
		return p.parseLabeledStmt()
	}
	if p.isDeclStart(p.curt) {
		return p.Decl(false)
	} else {
		switch p.curt.Kind {
		case cpp.CASE:
			return p.Case()
		case cpp.DEFAULT:
			return p.Default()
		case cpp.GOTO:
			return p.parseGoto()
		case ';':
			pos := p.curt.Pos
			p.next()
			return &EmptyStmt{
				Pos: pos,
			}
		case cpp.SWITCH:
			return p.Switch()
		case cpp.RETURN:
			return p.Return()
		case cpp.WHILE:
			return p.While()
		case cpp.DO:
			return p.DoWhile()
		case cpp.FOR:
			return p.For()
		case cpp.BREAK, cpp.CONTINUE:
			return p.BreakCont()
		case cpp.IF:
			return p.If()
		case '{':
			return p.Block()
		default:
			pos := p.curt.Pos
			expr := p.Expr()
			p.expect(';')
			return &ExprStmt{
				Pos:  pos,
				Expr: expr,
			}
		}
	}
	panic("unreachable.")
}

func (p *parser) Switch() Node {
	sw := &Switch{}
	sw.Pos = p.curt.Pos
	sw.LAfter = p.nextLabel()
	p.expect(cpp.SWITCH)
	p.expect('(')
	expr := p.Expr()
	sw.Expr = expr
	if !IsIntType(expr.GetType()) {
		p.errorPos(expr.GetPos(), "switch expression expects an integral type")
	}
	p.expect(')')
	p.pushSwitch(sw)
	p.pushBreak(sw.LAfter)
	stmt := p.Stmt()
	sw.Stmt = stmt
	p.popBreak()
	p.popSwitch()
	return sw
}

func (p *parser) parseGoto() Node {
	pos := p.curt.Pos
	p.next()
	actualLabel := p.curt.Val
	p.expect(cpp.IDENT)
	p.expect(';')
	ret := &Goto{
		Pos:   pos,
		Label: "", // To be fixed later.
	}
	p.gotos = append(p.gotos, gotoFixup{
		actualLabel,
		ret,
	})
	return ret
}

func (p *parser) parseLabeledStmt() Node {
	pos := p.curt.Pos
	label := p.curt.Val
	anonlabel := p.nextLabel()
	_, ok := p.labels[label]
	if ok {
		p.errorPos(pos, "redefinition of label %s in function", label)
	}
	p.labels[label] = anonlabel
	p.expect(cpp.IDENT)
	p.expect(':')
	return &LabeledStmt{
		Pos:       pos,
		Label:     label,
		AnonLabel: anonlabel,
		Stmt:      p.Stmt(),
	}
}

func (p *parser) Case() Node {
	pos := p.curt.Pos
	p.expect(cpp.CASE)
	sw := p.getSwitch()
	if sw == nil {
		p.errorPos(pos, "'case' outside a switch statement")
	}
	expr := p.Expr()
	if !IsIntType(expr.GetType()) {
		p.errorPos(expr.GetPos(), "expected an integral type")
	}
	v, err := p.fold(expr)
	if err != nil {
		p.errorPos(expr.GetPos(), err.Error())
	}
	p.expect(':')
	anonlabel := p.nextLabel()
	i := v.(*Constant)
	// XXX TODO
	swc := SwitchCase{
		V:     i.Val,
		Label: anonlabel,
	}
	sw.Cases = append(sw.Cases, swc)
	return &LabeledStmt{
		Pos:       pos,
		AnonLabel: anonlabel,
		Stmt:      p.Stmt(),
		IsCase:    true,
	}
}

func (p *parser) Default() Node {
	pos := p.curt.Pos
	p.expect(cpp.DEFAULT)
	sw := p.getSwitch()
	if sw == nil {
		p.errorPos(pos, "'default' outside a switch statement")
	}
	p.expect(':')
	if sw.LDefault != "" {
		p.errorPos(pos, "multiple default statements in switch")
	}
	anonlabel := p.nextLabel()
	sw.LDefault = anonlabel
	return &LabeledStmt{
		Pos:       pos,
		AnonLabel: anonlabel,
		Stmt:      p.Stmt(),
		IsDefault: true,
	}
}

func (p *parser) BreakCont() Node {
	pos := p.curt.Pos
	label := ""
	isbreak := p.curt.Kind == cpp.BREAK
	iscont := p.curt.Kind == cpp.CONTINUE
	if isbreak {
		label = p.getBreakLabel()
		if label == "" {
			p.errorPos(pos, "break outside of loop/switch")
		}
	}
	if iscont {
		label = p.getContLabel()
		if label == "" {
			p.errorPos(pos, "continue outside of loop/switch")
		}
	}
	p.next()
	p.expect(';')
	return &Goto{
		Pos:     pos,
		IsBreak: isbreak,
		IsCont:  iscont,
		Label:   label,
	}
}

func (p *parser) Return() Node {
	pos := p.curt.Pos
	p.expect(cpp.RETURN)
	expr := p.Expr()
	p.expect(';')
	return &Return{
		Pos: pos,
		Ret: expr,
	}
}

func (p *parser) If() Node {
	ifpos := p.curt.Pos
	lelse := p.nextLabel()
	p.expect(cpp.IF)
	p.expect('(')
	expr := p.Expr()
	p.ensureScalar(expr)
	p.expect(')')
	stmt := p.Stmt()
	var els Node
	if p.curt.Kind == cpp.ELSE {
		p.next()
		els = p.Stmt()
	}
	return &If{
		Pos:   ifpos,
		Cond:  expr,
		Stmt:  stmt,
		Else:  els,
		LElse: lelse,
	}
}

func (p *parser) For() Node {
	pos := p.curt.Pos
	lstart := p.nextLabel()
	lend := p.nextLabel()
	var init, cond, step Expr
	p.expect(cpp.FOR)
	p.expect('(')
	if p.curt.Kind != ';' {
		init = p.Expr()
	}
	p.expect(';')
	if p.curt.Kind != ';' {
		cond = p.Expr()
	}
	p.expect(';')
	if p.curt.Kind != ')' {
		step = p.Expr()
	}
	p.expect(')')
	p.pushBreakCont(lend, lstart)
	body := p.Stmt()
	p.popBreakCont()
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

func (p *parser) While() Node {
	pos := p.curt.Pos
	lstart := p.nextLabel()
	lend := p.nextLabel()
	p.expect(cpp.WHILE)
	p.expect('(')
	cond := p.Expr()
	p.ensureScalar(cond)
	p.expect(')')
	p.pushBreakCont(lend, lstart)
	body := p.Stmt()
	p.popBreakCont()
	return &While{
		Pos:    pos,
		Cond:   cond,
		Body:   body,
		LStart: lstart,
		LEnd:   lend,
	}
}

func (p *parser) DoWhile() Node {
	pos := p.curt.Pos
	lstart := p.nextLabel()
	lcond := p.nextLabel()
	lend := p.nextLabel()
	p.expect(cpp.DO)
	p.pushBreakCont(lend, lcond)
	body := p.Stmt()
	p.popBreakCont()
	p.expect(cpp.WHILE)
	p.expect('(')
	cond := p.Expr()
	p.expect(')')
	p.expect(';')
	return &DoWhile{
		Pos:    pos,
		Body:   body,
		Cond:   cond,
		LStart: lstart,
		LCond:  lcond,
		LEnd:   lend,
	}
}

func (p *parser) Block() *Block {
	var stmts []Node
	pos := p.curt.Pos
	p.expect('{')
	for p.curt.Kind != '}' {
		stmts = append(stmts, p.Stmt())
	}
	p.expect('}')
	return &Block{
		Pos:  pos,
		Body: stmts,
	}
}

func (p *parser) FuncBody(f *Function) {
	p.labels = make(map[string]string)
	p.gotos = nil
	for p.curt.Kind != '}' {
		stmt := p.Stmt()
		f.Body = append(f.Body, stmt)
	}
	for _, fixup := range p.gotos {
		anonlabel, ok := p.labels[fixup.actualLabel]
		if !ok {
			p.errorPos(fixup.g.GetPos(), "goto target %s is undefined", fixup.actualLabel)
		}
		fixup.g.Label = anonlabel
	}
}

func (p *parser) Decl(isGlobal bool) Node {
	firstDecl := true
	declPos := p.curt.Pos
	var name *cpp.Token
	declList := &DeclList{}
	sc, ty := p.DeclSpecs()
	declList.Storage = sc
	isTypedef := sc == SC_TYPEDEF

	if p.curt.Kind == ';' {
		p.next()
		return declList
	}

	for {
		name, ty = p.Declarator(ty, false)
		if name == nil {
			panic("internal error")
		}
		if firstDecl && isGlobal {
			// if declaring a function
			if p.curt.Kind == '{' {
				if isTypedef {
					p.errorPos(name.Pos, "cannot typedef a function")
				}
				fty, ok := ty.(*FunctionType)
				if !ok {
					p.errorPos(name.Pos, "expected a function")
				}
				err := p.decls.define(name.Val, &GSymbol{
					Label: name.Val,
					Type:  fty,
				})
				if err != nil {
					p.errorPos(declPos, err.Error())
				}
				p.pushScope()
				var psyms []*LSymbol

				for idx, name := range fty.ArgNames {
					sym := &LSymbol{
						Type: fty.ArgTypes[idx],
					}
					psyms = append(psyms, sym)
					err := p.decls.define(name, sym)
					if err != nil {
						p.errorPos(declPos, "multiple params with name %s", name)
					}
				}
				f := &Function{
					Name:         name.Val,
					FuncType:     fty,
					Pos:          declPos,
					ParamSymbols: psyms,
				}
				p.expect('{')
				p.FuncBody(f)
				p.expect('}')
				p.popScope()
				return f
			}
		}
		var sym Symbol
		if isTypedef {
			sym = &TSymbol{
				Type: ty,
			}
		} else if isGlobal {
			sym = &GSymbol{
				Label: name.Val,
				Type:  ty,
			}
		} else {
			sym = &LSymbol{
				Type: ty,
			}
		}
		var err error
		if isTypedef {
			err = p.types.define(name.Val, sym)
		} else {
			err = p.decls.define(name.Val, sym)
		}
		if err != nil {
			p.errorPos(name.Pos, err.Error())
		}
		declList.Symbols = append(declList.Symbols, sym)
		var init Expr
		var initPos cpp.FilePos
		if p.curt.Kind == '=' {
			p.next()
			initPos = p.curt.Pos
			if isTypedef {
				p.errorPos(initPos, "cannot initialize a typedef")
			}
			init = p.Initializer(ty, true)
		}
		declList.Inits = append(declList.Inits, init)
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

func (p *parser) ParamDecl() (*cpp.Token, CType) {
	_, ty := p.DeclSpecs()
	return p.Declarator(ty, true)
}

func isStorageClass(k cpp.TokenKind) (bool, SClass) {
	switch k {
	case cpp.STATIC:
		return true, SC_STATIC
	case cpp.EXTERN:
		return true, SC_GLOBAL
	case cpp.TYPEDEF:
		return true, SC_TYPEDEF
	case cpp.REGISTER:
		return true, SC_REGISTER
	}
	return false, 0
}

type dSpec struct {
	signedcnt   int
	unsignedcnt int
	charcnt     int
	intcnt      int
	shortcnt    int
	longcnt     int
	floatcnt    int
	doublecnt   int
}

type dSpecLutEnt struct {
	spec dSpec
	ty   Primitive
}

var declSpecLut = [...]dSpecLutEnt{
	{dSpec{
		charcnt: 1,
	}, CChar},
	{dSpec{
		signedcnt: 1,
		charcnt:   1,
	}, CChar},
	{dSpec{
		unsignedcnt: 1,
		charcnt:     1,
	}, CUChar},
	{dSpec{
		shortcnt: 1,
	}, CShort},
	{dSpec{
		signedcnt: 1,
		shortcnt:  1,
	}, CShort},
	{dSpec{
		intcnt:   1,
		shortcnt: 1,
	}, CShort},
	{dSpec{
		signedcnt: 1,
		intcnt:    1,
		shortcnt:  1,
	}, CShort},
	{dSpec{
		signedcnt: 1,
		intcnt:    1,
		shortcnt:  1,
	}, CShort},
	{dSpec{
		unsignedcnt: 1,
		intcnt:      1,
		shortcnt:    1,
	}, CUShort},
	{dSpec{
		unsignedcnt: 1,
		shortcnt:    1,
	}, CUShort},
	{dSpec{
		intcnt: 1,
	}, CInt},
	{dSpec{
		intcnt: 1,
	}, CInt},
	{dSpec{
		signedcnt: 1,
	}, CInt},
	{dSpec{
		signedcnt: 1,
		intcnt:    1,
	}, CInt},
	{dSpec{
		unsignedcnt: 1,
	}, CUInt},
	{dSpec{
		unsignedcnt: 1,
		intcnt:      1,
	}, CUInt},
	{dSpec{
		longcnt: 1,
	}, CLong},
	{dSpec{
		signedcnt: 1,
		longcnt:   1,
	}, CLong},
	{dSpec{
		longcnt: 1,
		intcnt:  1,
	}, CLong},
	{dSpec{
		signedcnt: 1,
		longcnt:   1,
		intcnt:    1,
	}, CLong},
	{dSpec{
		unsignedcnt: 1,
		longcnt:     1,
	}, CULong},
	{dSpec{
		unsignedcnt: 1,
		longcnt:     1,
		intcnt:      1,
	}, CLong},
	{dSpec{
		longcnt: 2,
	}, CLLong},
	{dSpec{
		signedcnt: 1,
		longcnt:   2,
	}, CLLong},
	{dSpec{
		intcnt:  1,
		longcnt: 2,
	}, CLLong},
	{dSpec{
		intcnt:    1,
		signedcnt: 1,
		longcnt:   2,
	}, CLLong},
	{dSpec{
		unsignedcnt: 1,
		longcnt:     2,
	}, CULLong},
	{dSpec{
		intcnt:      1,
		unsignedcnt: 1,
		longcnt:     2,
	}, CULLong},
	{dSpec{
		floatcnt: 1,
	}, CFloat},
	{dSpec{
		doublecnt: 1,
	}, CDouble},
}

func (p *parser) DeclSpecs() (SClass, CType) {
	dspecpos := p.curt.Pos
	scassigned := false
	sc := SC_AUTO
	var ty CType = CInt
	var spec dSpec
	nullspec := dSpec{}
loop:
	for {
		pos := p.curt.Pos
		issc, sclass := isStorageClass(p.curt.Kind)
		if issc {
			if scassigned {
				p.errorPos(pos, "only one storage class specifier allowed")
			}
			scassigned = true
			sc = sclass
			p.next()
			continue
		}
		switch p.curt.Kind {
		case cpp.VOID:
			p.next()
		case cpp.CHAR:
			spec.charcnt += 1
			p.next()
		case cpp.SHORT:
			spec.shortcnt += 1
			p.next()
		case cpp.INT:
			spec.intcnt += 1
			p.next()
		case cpp.LONG:
			spec.longcnt += 1
			p.next()
		case cpp.FLOAT:
			spec.floatcnt += 1
			p.next()
		case cpp.DOUBLE:
			spec.doublecnt += 1
			p.next()
		case cpp.SIGNED:
			spec.signedcnt += 1
			p.next()
		case cpp.UNSIGNED:
			spec.unsignedcnt += 1
			p.next()
		case cpp.IDENT:
			t := p.curt
			sym, err := p.types.lookup(t.Val)
			if err != nil {
				break loop
			}
			tsym := sym.(*TSymbol)
			p.next()
			if spec != nullspec {
				p.error("TODO...")
			}
			return sc, tsym.Type
		case cpp.STRUCT:
			if spec != nullspec {
				p.error("TODO...")
			}
			ty = p.Struct()
			return sc, ty
		case cpp.UNION:
		case cpp.VOLATILE, cpp.CONST:
			p.next()
		default:
			break loop
		}
	}

	// If we got any type specifiers, look up
	// the correct type.
	if spec != nullspec {
		match := false
		for _, te := range declSpecLut {
			if te.spec == spec {
				ty = te.ty
				match = true
				break
			}
		}
		if !match {
			p.errorPos(dspecpos, "invalid type")
		}
	}
	return sc, ty
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

func (p *parser) Declarator(basety CType, abstract bool) (*cpp.Token, CType) {
	for p.curt.Kind == cpp.CONST || p.curt.Kind == cpp.VOLATILE {
		p.next()
	}
	switch p.curt.Kind {
	case '*':
		p.next()
		name, ty := p.Declarator(basety, abstract)
		return name, &Ptr{ty}
	case '(':
		forward := &ForwardedType{}
		p.next()
		name, ty := p.Declarator(forward, abstract)
		p.expect(')')
		forward.Type = p.DeclaratorTail(basety)
		return name, ty
	case cpp.IDENT:
		name := p.curt
		p.next()
		return name, p.DeclaratorTail(basety)
	default:
		if abstract {
			return nil, p.DeclaratorTail(basety)
		}
		p.errorPos(p.curt.Pos, "expected ident, '(' or '*' but got %s", p.curt.Kind)
	}
	panic("unreachable")
}

func (p *parser) DeclaratorTail(basety CType) CType {
	ret := basety
	for {
		switch p.curt.Kind {
		case '[':
			p.next()
			var dimn Expr
			if p.curt.Kind != ']' {
				dimn = p.AssignmentExpr()
			}
			p.expect(']')
			dim, err := p.fold(dimn)
			if err != nil {
				p.errorPos(dimn.GetPos(), "invalid constant Expr for array dimensions")
			}
			i, ok := dim.(*Constant)
			if !ok || !IsIntType(i.Type) {
				p.errorPos(dimn.GetPos(), "Expected an int type for array length")
			}
			ret = &Array{
				Dim:        int(i.Val),
				MemberType: ret,
			}
		case '(':
			fret := &FunctionType{}
			fret.RetType = basety
			p.next()
			if p.curt.Kind != ')' {
				for {
					pnametok, pty := p.ParamDecl()
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

func (p *parser) Initializer(ty CType, constant bool) Expr {
	_ = p.curt.Pos
	if IsScalarType(ty) {
		var init Expr
		if p.curt.Kind == '{' {
			p.expect('{')
			init = p.AssignmentExpr()
			p.expect('}')
		} else {
			init = p.AssignmentExpr()
		}
		// XXX ensure types are compatible.
		// XXX Add cast.
		if constant {
			c, err := p.fold(init)
			if err != nil {
				p.errorPos(init.GetPos(), err.Error())
			}
			return c
		} else {
			return init
		}
	} /* else if IsCharArr(ty) {
		switch p.curt.Kind {
		case cpp.STRING:
			p.expect(cpp.STRING)
		case '{':
			p.expect('{')
			p.expect(cpp.STRING)
			p.expect('}')
		default:
		}
	} else if IsArrType(ty) {
		arr := ty.(*Array)
		p.expect('{')
		var inits []Node
		for p.curt.Kind != '}' {
			inits = append(inits, p.parseInitializer(arr.MemberType, true))
			if p.curt.Kind == ',' {
				continue
			}
		}
		p.expect('}')
	}
	*/
	panic("unimplemented")
}

func isAssignmentOperator(k cpp.TokenKind) bool {
	switch k {
	case '=', cpp.ADD_ASSIGN, cpp.SUB_ASSIGN, cpp.MUL_ASSIGN, cpp.QUO_ASSIGN, cpp.REM_ASSIGN,
		cpp.AND_ASSIGN, cpp.OR_ASSIGN, cpp.XOR_ASSIGN, cpp.SHL_ASSIGN, cpp.SHR_ASSIGN:
		return true
	}
	return false
}

func (p *parser) Expr() Expr {
	var ret Expr
	for {
		ret = p.AssignmentExpr()
		if p.curt.Kind != ',' {
			break
		}
		p.next()
	}
	return ret
}

func (p *parser) AssignmentExpr() Expr {
	l := p.CondExpr()
	if isAssignmentOperator(p.curt.Kind) {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.AssignmentExpr()
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

// Aka Ternary operator.
func (p *parser) CondExpr() Expr {
	return p.LogOrExpr()
}

func (p *parser) LogOrExpr() Expr {
	l := p.LogAndExpr()
	for p.curt.Kind == cpp.LOR {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.LogAndExpr()
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

func (p *parser) LogAndExpr() Expr {
	l := p.OrExpr()
	for p.curt.Kind == cpp.LAND {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.OrExpr()
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

func (p *parser) OrExpr() Expr {
	l := p.XorExpr()
	for p.curt.Kind == '|' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.XorExpr()
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

func (p *parser) XorExpr() Expr {
	l := p.AndExpr()
	for p.curt.Kind == '^' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.AndExpr()
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

func (p *parser) AndExpr() Expr {
	l := p.EqlExpr()
	for p.curt.Kind == '&' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.EqlExpr()
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

func (p *parser) EqlExpr() Expr {
	l := p.RelExpr()
	for p.curt.Kind == cpp.EQL || p.curt.Kind == cpp.NEQ {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.RelExpr()
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

func (p *parser) RelExpr() Expr {
	l := p.ShiftExpr()
	for p.curt.Kind == '>' || p.curt.Kind == '<' || p.curt.Kind == cpp.LEQ || p.curt.Kind == cpp.GEQ {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.ShiftExpr()
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

func (p *parser) ShiftExpr() Expr {
	l := p.AddExpr()
	for p.curt.Kind == cpp.SHL || p.curt.Kind == cpp.SHR {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.AddExpr()
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

func (p *parser) AddExpr() Expr {
	l := p.MulExpr()
	for p.curt.Kind == '+' || p.curt.Kind == '-' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.MulExpr()
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

func (p *parser) MulExpr() Expr {
	l := p.CastExpr()
	for p.curt.Kind == '*' || p.curt.Kind == '/' || p.curt.Kind == '%' {
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		r := p.CastExpr()
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

func (p *parser) CastExpr() Expr {
	// Cast
	if p.curt.Kind == '(' {
		if p.isDeclStart(p.nextt) {
			pos := p.curt.Pos
			p.expect('(')
			ty := p.TypeName()
			p.expect(')')
			operand := p.UnaryExpr()
			return &Cast{
				Pos:     pos,
				Operand: operand,
				Type:    ty,
			}
		}
	}
	return p.UnaryExpr()
}

func (p *parser) TypeName() CType {
	_, ty := p.DeclSpecs()
	_, ty = p.Declarator(ty, true)
	return ty
}

func (p *parser) UnaryExpr() Expr {
	switch p.curt.Kind {
	case cpp.INC, cpp.DEC:
		p.next()
		p.UnaryExpr()
	case '*', '+', '-', '!', '~', '&':
		pos := p.curt.Pos
		op := p.curt.Kind
		p.next()
		operand := p.CastExpr()
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
		return p.PostExpr()
	}
	panic("unreachable")
}

func (p *parser) PostExpr() Expr {
	l := p.PrimaryExpr()
loop:
	for {
		switch p.curt.Kind {
		case '[':
			var ty CType
			arr, isArr := l.GetType().(*Array)
			ptr, isPtr := l.GetType().(*Ptr)
			if !isArr && !isPtr {
				p.errorPos(p.curt.Pos, "Can only index into array or pointer types")
			}
			if isArr {
				ty = arr.MemberType
			}
			if isPtr {
				ty = ptr.PointsTo
			}
			p.next()
			idx := p.Expr()
			p.expect(']')
			l = &Index{
				Arr:  l,
				Idx:  idx,
				Type: ty,
			}
		case '.', cpp.ARROW:
			op := p.curt.Kind
			pos := p.curt.Pos
			strct, isStruct := l.GetType().(*CStruct)
			p.next()
			if !isStruct {
				p.errorPos(l.GetPos(), "expected a struct")
			}
			sel := p.curt
			p.expect(cpp.IDENT)
			ty := strct.FieldType(sel.Val)
			if ty == nil {
				p.errorPos(pos, "struct does not have field %s", sel.Val)
			}
			l = &Selector{
				Op:      op,
				Pos:     pos,
				Operand: l,
				Type:    ty,
				Sel:     sel.Val,
			}
		case '(':
			parenpos := p.curt.Pos
			var fty *FunctionType
			switch ty := l.GetType().(type) {
			case *Ptr:
				functy, ok := ty.PointsTo.(*FunctionType)
				if !ok {
					p.errorPos(l.GetPos(), "expected a function pointer")
				}
				fty = functy
			case *FunctionType:
				fty = ty
			default:
				p.errorPos(l.GetPos(), "expected a func or func pointer")
			}
			var args []Expr
			p.next()
			if p.curt.Kind != ')' {
				for {
					args = append(args, p.AssignmentExpr())
					if p.curt.Kind == ',' {
						p.next()
						continue
					}
					break
				}
			}
			p.expect(')')
			return &Call{
				Pos:      parenpos,
				FuncLike: l,
				Args:     args,
				Type:     fty.RetType,
			}
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

func (p *parser) PrimaryExpr() Expr {
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
		s := p.curt
		p.next()
		l := p.nextLabel()
		rstr := &String{
			Pos:   s.Pos,
			Val:   s.Val,
			Label: l,
		}
		p.addAnonymousString(rstr)
		return rstr
	case '(':
		p.next()
		expr := p.Expr()
		p.expect(')')
		return expr
	default:
		p.errorPos(p.curt.Pos, "expected an identifier, constant, string or Expr")
	}
	panic("unreachable")
}

func (p *parser) Struct() CType {
	p.expect(cpp.STRUCT)
	var ret *CStruct
	sname := ""
	npos := p.curt.Pos
	if p.curt.Kind == cpp.IDENT {
		sname = p.curt.Val
		p.next()
		sym, err := p.structs.lookup(sname)
		if err != nil && p.curt.Kind != '{' {
			p.errorPos(npos, err.Error())
		}
		if err == nil {
			ret = sym.(*TSymbol).Type.(*CStruct)
		}
	}
	if p.curt.Kind == '{' {
		p.expect('{')
		ret = &CStruct{}
		for {
			if p.curt.Kind == '}' {
				break
			}
			_, basety := p.DeclSpecs()
			for {
				name, ty := p.Declarator(basety, false)
				ret.Names = append(ret.Names, name.Val)
				ret.Types = append(ret.Types, ty)
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
	if sname != "" {
		// TODO:
		// If ret is nil, is this a predefine?
		// Do we need an incomplete type?
		err := p.structs.define(sname, &TSymbol{
			Type: ret,
		})
		if err != nil {
			p.errorPos(npos, err.Error())
		}
	}
	return ret
}
