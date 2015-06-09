package main

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"github.com/andrewchambers/cc/parse"
	"io"
)

type emitter struct {
	o            io.Writer
	labelcounter int
	loffsets     map[*parse.LSymbol]int
	f            *parse.Function
}

func (e *emitter) NextLabel() string {
	e.labelcounter += 1
	return fmt.Sprintf(".LL%d", e.labelcounter)
}

func Emit(tu *parse.TranslationUnit, o io.Writer) error {
	e := &emitter{
		o: o,
	}

	for _, init := range tu.AnonymousInits {
		switch init := init.(type) {
		case *parse.String:
			e.emit(".data\n")
			e.emit("%s:\n", init.Label)
			e.emit(".string %s\n", init.Val)
		default:
			panic(init)
		}
	}

	for _, tl := range tu.TopLevels {
		switch tl := tl.(type) {
		case *parse.Function:
			e.emitFunction(tl)
		case *parse.DeclList:
			if tl.Storage == parse.SC_TYPEDEF {
				continue
			}
			for idx, decl := range tl.Symbols {
				global, ok := decl.(*parse.GSymbol)
				if !ok {
					panic("internal error")
				}
				e.emitGlobal(global, tl.Inits[idx])
			}
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

func (e *emitter) emitScalarExtend(sz int, signed bool) {
	if signed {
		switch sz {
		case 8:
			// Nothing to do.
		case 4:
			e.emiti("movsdq %%eax, %%rax\n")
		case 2:
			e.emiti("movswq %%ax, %%rax\n")
		case 1:
			e.emiti("movsbq %%al, %%rax\n")
		default:
			panic("internal error")
		}
	} else {
		switch sz {
		case 8:
			// Nothing to do.
		case 4:
			// *NOTE* This zeros top half of rax.
			e.emiti("mov %%eax, %%eax\n")
			return
		case 2:
			e.emiti("movzwq %%ax, %%eax\n")
			return
		case 1:
			e.emiti("movzbq %%al, %%eax\n")
			return
		default:
	        panic("internal error")
		}
	}
}

func (e *emitter) emitScalarLoadFromPtr(reg string, sz int, signed bool) {
	if signed {
		switch sz {
		case 8:
			e.emiti("movq (%%%s), %%rax\n", reg)
		case 4:
			e.emiti("movslq (%%%s), %%rax\n", reg)
		case 2:
			e.emiti("movswq (%%%s), %%rax\n", reg)
		case 1:
			e.emiti("movsbq (%%%s), %%rax\n", reg)
		default:
	        panic("internal error")
		}
	} else {
		switch sz {
		case 8:
			e.emiti("movq (%%%s), %%rax\n", reg)
		case 4:
			e.emiti("movzlq (%%%s), %%rax\n", reg)
		case 2:
			e.emiti("movzwq (%%%s), %%rax\n", reg)
		case 1:
			e.emiti("movzbq (%%%s), %%rax\n", reg)
	    default:
	        panic("internal error")
		}
	}
}

func (e *emitter) emitScalarStoreToPtr(reg string, sz int) {
	switch sz {
	case 8:
		e.emiti("movq %%rax, (%%%s)\n", reg)
	case 4:
		e.emiti("movl %%eax, (%%%s)\n", reg)
	case 2:
		e.emiti("movw %%ax, (%%%s)\n", reg)
	case 1:
		e.emiti("movb %%al, (%%%s)\n", reg)
	default:
	    panic("internal error")
	}
}

func (e *emitter) emitLoadGlobal(label string, ty parse.CType) {
	switch {
	case parse.IsCFunction(ty):
		e.emiti("leaq %s(%%rip), %%rax\n", label)
	case parse.IsIntType(ty):
		e.emiti("leaq %s(%%rip), %%rcx\n", label)
		if parse.IsSignedIntType(ty) {
			e.emitScalarLoadFromPtr("rcx", getSize(ty), true)
		} else {
			e.emitScalarLoadFromPtr("rcx", getSize(ty), false)
		}
	case parse.IsPtrType(ty):
		e.emiti("leaq %s(%%rip), %%rcx\n", label)
		e.emitScalarLoadFromPtr("rcx", getSize(ty), false)
	case parse.IsArrType(ty):
		e.emiti("leaq %s(%%rip), %%rax\n", label)
	default:
		panic(ty)
	}
}

func (e *emitter) emitLoadLocal(offset int, ty parse.CType) {
	switch {
	case parse.IsIntType(ty):
		e.emiti("leaq %d(%%rbp), %%rcx\n", offset)
		if parse.IsSignedIntType(ty) {
			e.emitScalarLoadFromPtr("rcx", getSize(ty), true)
		} else {
			e.emitScalarLoadFromPtr("rcx", getSize(ty), false)
		}
	case parse.IsPtrType(ty):
		e.emiti("leaq %d(%%rbp), %%rcx\n", offset)
		e.emitScalarLoadFromPtr("rcx", getSize(ty), false)
	case parse.IsArrType(ty):
		e.emiti("leaq %d(%%rbp), %%rax\n", offset)
	case parse.IsStruct(ty):
		e.emiti("leaq %d(%%rbp), %%rax\n", offset)
	default:
		panic(ty)
	}
}

func (e *emitter) emitLoadPtr(ty parse.CType) {
	switch {
	case parse.IsIntType(ty):
		if parse.IsSignedIntType(ty) {
			e.emitScalarLoadFromPtr("rax", getSize(ty), true)
		} else {
			e.emitScalarLoadFromPtr("rax", getSize(ty), false)
		}
	case parse.IsPtrType(ty):
		e.emitScalarLoadFromPtr("rax", getSize(ty), false)
	case parse.IsArrType(ty):
	case parse.IsStruct(ty):
	default:
		panic(ty)
	}
}


func (e *emitter) emitStoreGlobal(label string, ty parse.CType) {
	switch {
	case parse.IsIntType(ty):
		e.emiti("leaq %s(%%rip), %%rcx\n", label)
		if parse.IsSignedIntType(ty) {
			e.emitScalarStoreToPtr("rcx", getSize(ty))
		} else {
			e.emitScalarStoreToPtr("rcx", getSize(ty))
		}
	case parse.IsPtrType(ty):
		e.emiti("leaq %s(%%rip), %%rcx\n", label)
		e.emitScalarStoreToPtr("rcx", getSize(ty))
	default:
		panic(ty)
	}
}

func (e *emitter) emitStoreLocal(offset int, ty parse.CType) {
	switch {
	case parse.IsIntType(ty):
		e.emiti("leaq %d(%%rbp), %%rcx\n", offset)
		if parse.IsSignedIntType(ty) {
			e.emitScalarStoreToPtr("rcx", getSize(ty))
		} else {
			e.emitScalarStoreToPtr("rcx", getSize(ty))
		}
	case parse.IsPtrType(ty):
		e.emiti("leaq %d(%%rbp), %%rcx\n", offset)
		e.emitScalarStoreToPtr("rcx", getSize(ty))
	default:
		panic(ty)
	}
}

func (e *emitter) emitStorePtr(reg string, ty parse.CType) {
	switch {
	case parse.IsIntType(ty):
		if parse.IsSignedIntType(ty) {
			e.emitScalarStoreToPtr(reg, getSize(ty))
		} else {
			e.emitScalarStoreToPtr(reg, getSize(ty))
		}
	case parse.IsPtrType(ty):
		e.emitScalarStoreToPtr(reg, getSize(ty))
	default:
		panic(ty)
	}
}

func (e *emitter) emitGlobal(g *parse.GSymbol, init parse.Expr) {
	_, ok := g.Type.(*parse.FunctionType)
	if ok {
		return
	}
	e.emit(".data\n")
	e.emit(".global %s\n", g.Label)
	if init == nil {
		e.emit(".lcomm %s, %d\n", g.Label, getSize(g.Type))
	} else {
		e.emit("%s:\n", g.Label)
		switch {
		case parse.IsIntType(g.Type):
			v := init.(*parse.Constant)
			switch getSize(g.Type) {
			case 8:
				e.emit(".quad %d\n", v.Val)
			case 4:
				e.emit(".long %d\n", v.Val)
			case 2:
				e.emit(".short %d\n", v.Val)
			case 1:
				e.emit(".byte %d\n", v.Val)
			}
		case parse.IsPtrType(g.Type):
			switch init := init.(type) {
			case *parse.ConstantGPtr:
				switch {
				case init.Offset > 0:
					e.emit(".quad %s + %d\n", init.Offset)
				case init.Offset < 0:
					e.emit(".quad %s - %d\n", init.Offset)
				default:
					e.emit(".quad %s\n", init.PtrLabel)
				}
			case *parse.String:
				e.emit(".quad %s\n", init.Label)
			}
		default:
			panic("unimplemented")
		}
	}
}

var intParamLUT = [...]string{
	"%rdi", "%rsi", "%rdx", "%rcx", "r8", "r9",
}

func (e *emitter) emitFunction(f *parse.Function) {
	e.f = f
	e.emit(".text\n")
	e.emit(".global %s\n", f.Name)
	e.emit("%s:\n", f.Name)
	e.emiti("pushq %%rbp\n")
	e.emiti("movq %%rsp, %%rbp\n")
	curlocaloffset, loffsets := e.calcLocalOffsets(f)
	e.loffsets = loffsets
	if curlocaloffset != 0 {
		e.emiti("sub $%d, %%rsp\n", -curlocaloffset)
	}
	for idx, psym := range f.ParamSymbols {
		e.emiti("movq %s, %d(%%rbp)\n", intParamLUT[idx], e.loffsets[psym])
	}
	for _, stmt := range f.Body {
		e.emitStmt(stmt)
	}
	e.emiti("leave\n")
	e.emiti("ret\n")
	e.f = nil
}

func (e *emitter) calcLocalOffsets(f *parse.Function) (int, map[*parse.LSymbol]int) {
	loffset := 0
	loffsets := make(map[*parse.LSymbol]int)
	addLSymbol := func(lsym *parse.LSymbol) {
		sz := getSize(lsym.Type)
		if sz < 8 {
			sz = 8
		}
		sz = sz + (sz % 8)
		loffset -= sz
		loffsets[lsym] = loffset
	}
	for _, lsym := range f.ParamSymbols {
		addLSymbol(lsym)
	}
	for _, n := range f.Body {
		switch n := n.(type) {
		case *parse.DeclList:
			for _, sym := range n.Symbols {
				lsym, ok := sym.(*parse.LSymbol)
				if !ok {
					continue
				}
				addLSymbol(lsym)
			}

		}
	}
	return loffset, loffsets
}

func (e *emitter) emitStmt(stmt parse.Node) {
	switch stmt := stmt.(type) {
	case *parse.If:
		e.emitIf(stmt)
	case *parse.While:
		e.emitWhile(stmt)
	case *parse.DoWhile:
		e.emitDoWhile(stmt)
	case *parse.For:
		e.emitFor(stmt)
	case *parse.Return:
		e.emitReturn(stmt)
	case *parse.CompndStmt:
		e.emitCompndStmt(stmt)
	case *parse.ExprStmt:
		e.emitExpr(stmt.Expr)
	case *parse.Goto:
		e.emiti("jmp %s\n", stmt.Label)
	case *parse.LabeledStmt:
		e.emit("%s:\n", stmt.AnonLabel)
		e.emitStmt(stmt.Stmt)
	case *parse.Switch:
		e.emitSwitch(stmt)
	case *parse.EmptyStmt:
		// pass
	case *parse.DeclList:
		// pass
	default:
		panic(stmt)
	}
}

func (e *emitter) emitSwitch(sw *parse.Switch) {
	e.emitExpr(sw.Expr)
	for _, swc := range sw.Cases {
		e.emiti("mov $%d, %%rcx\n", swc.V)
		e.emiti("cmp %%rax, %%rcx\n")
		e.emiti("je %s\n", swc.Label)
	}
	if sw.LDefault != "" {
		e.emiti("jmp %s\n", sw.LDefault)
	} else {
		e.emiti("jmp %s\n", sw.LAfter)
	}
	e.emitStmt(sw.Stmt)
	e.emit("%s:\n", sw.LAfter)
}

func (e *emitter) emitWhile(w *parse.While) {
	e.emit("%s:\n", w.LStart)
	e.emitExpr(w.Cond)
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", w.LEnd)
	e.emitStmt(w.Body)
	e.emiti("jmp %s\n", w.LStart)
	e.emit("%s:\n", w.LEnd)
}

func (e *emitter) emitDoWhile(d *parse.DoWhile) {
	e.emit("%s:\n", d.LStart)
	e.emitStmt(d.Body)
	e.emit("%s:\n", d.LCond)
	e.emitExpr(d.Cond)
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", d.LEnd)
	e.emiti("jmp %s\n", d.LStart)
	e.emit("%s:\n", d.LEnd)
}

func (e *emitter) emitFor(fr *parse.For) {
	if fr.Init != nil {
		e.emitExpr(fr.Init)
	}
	e.emit("%s:\n", fr.LStart)
	if fr.Cond != nil {
		e.emitExpr(fr.Cond)
	}
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", fr.LEnd)
	e.emitStmt(fr.Body)
	if fr.Step != nil {
		e.emitExpr(fr.Step)
	}
	e.emiti("jmp %s\n", fr.LStart)
	e.emit("%s:\n", fr.LEnd)
}

func (e *emitter) emitCompndStmt(c *parse.CompndStmt) {
	for _, stmt := range c.Body {
		e.emitStmt(stmt)
	}
}

func (e *emitter) emitIf(i *parse.If) {
	e.emitExpr(i.Cond)
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", i.LElse)
	e.emitStmt(i.Stmt)
	e.emit("%s:\n", i.LElse)
	if i.Else != nil {
		e.emitStmt(i.Else)
	}
}

func (e *emitter) emitReturn(r *parse.Return) {
	e.emitExpr(r.Ret)
	e.emiti("leave\n")
	e.emiti("ret\n")
}

func (e *emitter) emitExpr(expr parse.Node) {
	switch expr := expr.(type) {
	case *parse.Ident:
		e.emitIdent(expr)
	case *parse.Call:
		e.emitCall(expr)
	case *parse.Constant:
		e.emiti("movq $%v, %%rax\n", expr.Val)
	case *parse.Unop:
		e.emitUnop(expr)
	case *parse.Binop:
		e.emitBinop(expr)
	case *parse.Index:
		e.emitIndex(expr)
	case *parse.Cast:
		e.emitCast(expr)
	case *parse.Selector:
		e.emitSelector(expr)
	case *parse.String:
		e.emiti("leaq %s(%%rip), %%rax\n", expr.Label)
	default:
		panic(expr)
	}
}

func getStructOffset(s *parse.CStruct, member string) int {
	offset := 0
	for idx, n := range s.Names {
		if n == member {
			return offset
		}
		offset += getSize(s.Types[idx])
	}
	// Error should have been caught in parse.
	panic("internal error")
}

func (e *emitter) emitSelector(s *parse.Selector) {
	e.emitExpr(s.Operand)
	ty := s.Operand.GetType()
	offset := 0
	switch ty := ty.(type) {
	case *parse.CStruct:
		offset = getStructOffset(ty, s.Sel)
	default:
		panic("internal error")
	}
	if offset != 0 {
		e.emiti("add $%d, %%rax\n", offset)
	}
	switch {
	case parse.IsIntType(s.Type):
		if parse.IsSignedIntType(s.Type) {
			e.emitScalarLoadFromPtr("rax", getSize(s.Type), true)
		} else {
			e.emitScalarLoadFromPtr("rax", getSize(s.Type), false)
		}
	case parse.IsPtrType(s.Type):
		e.emitScalarLoadFromPtr("rax", getSize(s.Type), false)
	default:
		panic(s.Type)
	}
}

func (e *emitter) emitIdent(i *parse.Ident) {
	sym := i.Sym
	switch sym := sym.(type) {
	case *parse.LSymbol:
		offset := e.loffsets[sym]
		e.emitLoadLocal(offset, sym.Type)
	case *parse.GSymbol:
		e.emitLoadGlobal(sym.Label, sym.Type)
	}
}

func isIntRegArg(t parse.CType) bool {
	return parse.IsIntType(t) || parse.IsPtrType(t)
}

func classifyArgs(args []parse.Expr) ([]parse.Expr, []parse.Expr) {
	var intargs, memargs []parse.Expr
	nintargs := 0
	for _, arg := range args {
		if nintargs < 6 && isIntRegArg(arg.GetType()) {
			nintargs += 1
			intargs = append(intargs, arg)
		} else {
			memargs = append(memargs, arg)
		}
	}
	return intargs, memargs
}

func (e *emitter) emitCall(c *parse.Call) {
	intargs, memargs := classifyArgs(c.Args)
	sz := 0
	for i := len(memargs) - 1; i >= 0; i-- {
		arg := memargs[i]
		e.emitExpr(arg)
		e.emiti("push %%rax\n")
		sz += 8
	}
	for i := len(intargs) - 1; i >= 0; i-- {
		arg := intargs[i]
		e.emitExpr(arg)
		e.emiti("push %%rax\n")
	}
	for idx, _ := range intargs {
		e.emiti("pop %s\n", intParamLUT[idx])
	}
	e.emitExpr(c.FuncLike)
	e.emiti("call *%%rax\n")
	if sz != 0 {
		e.emiti("add $%d, %%rsp\n", sz)
	}
}

func (e *emitter) emitCast(c *parse.Cast) {
	e.emitExpr(c.Operand)
	from := c.Operand.GetType()
	to := c.Type
	switch {
	case parse.IsPtrType(to):
		if parse.IsPtrType(from) || parse.IsIntType(from) {
			return
		}
	case parse.IsIntType(to):
		if parse.IsPtrType(from) || parse.IsIntType(from) {
			return
		}
	}
	panic("unimplemented cast")
}

func (e *emitter) emitBinop(b *parse.Binop) {
	if b.Op == '=' {
		e.emitAssign(b)
		return
	}
	e.emitExpr(b.L)
	e.emiti("pushq %%rax\n")
	e.emitExpr(b.R)
	e.emiti("movq %%rax, %%rcx\n")
	e.emiti("popq %%rax\n")
	switch {
	case parse.IsIntType(b.Type):
		switch b.Op {
		case '+':
			e.emiti("addq %%rcx, %%rax\n")
		case '-':
			e.emiti("subq %%rcx, %%rax\n")
		case '*':
			e.emiti("imul %%rcx, %%rax\n")
		case '|':
			e.emiti("or %%rcx, %%rax\n")
		case '&':
			e.emiti("and %%rcx, %%rax\n")
		case '^':
			e.emiti("xor %%rcx, %%rax\n")
		case '/':
			e.emiti("cqto\n")
			e.emiti("idiv %%rcx\n")
		case '%':
			e.emiti("idiv %%rcx\n")
			e.emiti("mov %%rdx, %%rax\n")
		case cpp.SHL:
			e.emiti("sal %%cl, %%rax\n")
		case cpp.SHR:
			e.emiti("sar %%cl, %%rax\n")
		case cpp.EQL, cpp.NEQ, '>', '<':
			lset := e.NextLabel()
			lafter := e.NextLabel()
			opc := ""
			switch b.Op {
			case cpp.EQL:
				opc = "jz"
			case cpp.NEQ:
				opc = "jnz"
			case '<':
				opc = "jl"
			case '>':
				opc = "jg"
			default:
				panic("internal error")
			}
			e.emiti("cmp %%rcx, %%rax\n")
			e.emiti("%s %s\n", opc, lset)
			e.emiti("movq $0, %%rax\n")
			e.emiti("jmp %s\n", lafter)
			e.emiti("%s:\n", lset)
			e.emiti("movq $1, %%rax\n")
			e.emiti("%s:\n", lafter)
		default:
			panic("unimplemented " + b.Op.String())
		}
	default:
		panic(b)
	}
}

func (e *emitter) emitUnop(u *parse.Unop) {
	switch u.Op {
	case '&':
		switch operand := u.Operand.(type) {
		case *parse.Unop:
			if operand.Op != '*' {
				panic("internal error")
			}
			e.emitExpr(operand.Operand)
		case *parse.Ident:
			sym := operand.Sym
			switch sym := sym.(type) {
			case *parse.GSymbol:
				e.emiti("leaq %s(%%rip), %%rax\n", sym.Label)
			case *parse.LSymbol:
				offset := e.loffsets[sym]
				e.emiti("leaq %d(%%rbp), %%rax\n", offset)
			default:
				panic("internal error")
			}
		case *parse.Index:
			e.emitExpr(operand.Idx)
			sz := getSize(operand.GetType())
			e.emiti("imul $%d, %%rax\n", sz)
			if sz != 1 {
				e.emiti("imul $%d, %%rax\n", sz)
			}
			e.emiti("push %%rax\n")
			e.emitExpr(operand.Arr)
			e.emiti("pop %%rcx\n")
			e.emiti("addq %%rcx, %%rax\n")
		default:
			panic("internal error")
		}
	case '!':
		e.emitExpr(u.Operand)
		e.emiti("xor %%rcx, %%rcx\n")
		switch getSize(u.GetType()) {
		case 8:
			e.emiti("test %%rax, %%rax\n")
		case 4:
			e.emiti("test %%eax, %%eax\n")
		}
		e.emiti("setnz %%cl\n")
		e.emiti("mov %%rcx, %%rax\n")
	case '-':
		e.emitExpr(u.Operand)
		e.emiti("neg %%rax\n")
	case '*':
		e.emitExpr(u.Operand)
		e.emiti("movq (%%rax), %%rax\n")
	}
}

func (e *emitter) emitIndex(idx *parse.Index) {
	e.emitExpr(idx.Idx)
	sz := getSize(idx.GetType())
	if sz != 1 {
		e.emiti("imul $%d, %%rax\n", sz)
	}
	e.emiti("push %%rax\n")
	e.emitExpr(idx.Arr)
	e.emiti("pop %%rcx\n")
	e.emiti("addq %%rcx, %%rax\n")
	e.emitLoadPtr(idx.GetType())
}

func (e *emitter) emitAssign(b *parse.Binop) {
	e.emitExpr(b.R)
	switch l := b.L.(type) {
	case *parse.Index:
		e.emiti("push %%rax\n")
		e.emitExpr(l.Idx)
		e.emiti("imul $%d, %%rax\n", 4)
		e.emiti("push %%rax\n")
		e.emitExpr(l.Arr)
		e.emiti("pop %%rcx\n")
		e.emiti("add %%rcx, %%rax\n")
		e.emiti("pop %%rcx\n")
		e.emitStorePtr("rcx", l.GetType())
	case *parse.Selector:
		e.emiti("push %%rax\n")
		e.emitExpr(l.Operand)
		ty := l.Operand.GetType()
		offset := 0
		switch ty := ty.(type) {
		case *parse.CStruct:
			offset = getStructOffset(ty, l.Sel)
		default:
			panic("internal error")
		}
		e.emiti("add $%d,%%rax\n", offset)
		e.emiti("pop %%rcx\n")
		e.emitStorePtr("rcx", l.GetType())
	case *parse.Unop:
		if l.Op != '*' {
			panic("internal error")
		}
		e.emiti("push %%rax\n")
		e.emitExpr(l.Operand)
		e.emiti("pop %%rcx\n")
		e.emitStorePtr("rcx", l.GetType())
	case *parse.Ident:
		sym := l.Sym
		switch sym := sym.(type) {
		case *parse.GSymbol:
			e.emitStoreGlobal(sym.Label, sym.Type)
		case *parse.LSymbol:
			offset := e.loffsets[sym]
			e.emitStoreLocal(offset, sym.Type)
		}
	default:
		panic(b.L)
	}
}
