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
			e.raw(".data\n")
			e.raw("%s:\n", init.Label)
			e.raw(".string %s\n", init.Val)
		default:
			panic(init)
		}
	}

	for _, tl := range tu.TopLevels {
		switch tl := tl.(type) {
		case *parse.Function:
			e.Function(tl)
		case *parse.DeclList:
			if tl.Storage == parse.SC_TYPEDEF {
				continue
			}
			for idx, decl := range tl.Symbols {
				global, ok := decl.(*parse.GSymbol)
				if !ok {
					panic("internal error")
				}
				e.Global(global, tl.Inits[idx])
			}
		default:
			panic(tl)
		}
	}
	return nil
}

func (e *emitter) raw(s string, args ...interface{}) {
	fmt.Fprintf(e.o, s, args...)
}

func (e *emitter) asm(s string, args ...interface{}) {
	e.raw("  "+s, args...)
}

func (e *emitter) Global(g *parse.GSymbol, init parse.Expr) {
	_, ok := g.Type.(*parse.FunctionType)
	if ok {
		return
	}
	e.raw(".data\n")
	e.raw(".global %s\n", g.Label)
	if init == nil {
		e.raw(".lcomm %s, %d\n", g.Label, getSize(g.Type))
	} else {
		e.raw("%s:\n", g.Label)
		switch {
		case parse.IsIntType(g.Type):
			v := init.(*parse.Constant)
			switch getSize(g.Type) {
			case 8:
				e.raw(".quad %d\n", v.Val)
			case 4:
				e.raw(".long %d\n", v.Val)
			case 2:
				e.raw(".short %d\n", v.Val)
			case 1:
				e.raw(".byte %d\n", v.Val)
			}
		case parse.IsPtrType(g.Type):
			switch init := init.(type) {
			case *parse.ConstantGPtr:
				switch {
				case init.Offset > 0:
					e.raw(".quad %s + %d\n", init.Offset)
				case init.Offset < 0:
					e.raw(".quad %s - %d\n", init.Offset)
				default:
					e.raw(".quad %s\n", init.PtrLabel)
				}
			case *parse.String:
				e.raw(".quad %s\n", init.Label)
			}
		default:
			panic("unimplemented")
		}
	}
}

var intParamLUT = [...]string{
	"%rdi", "%rsi", "%rdx", "%rcx", "r8", "r9",
}

func (e *emitter) Function(f *parse.Function) {
	e.f = f
	e.raw(".text\n")
	e.raw(".global %s\n", f.Name)
	e.raw("%s:\n", f.Name)
	e.asm("pushq %%rbp\n")
	e.asm("movq %%rsp, %%rbp\n")
	curlocaloffset, loffsets := e.calcLocalOffsets(f)
	e.loffsets = loffsets
	if curlocaloffset != 0 {
		e.asm("sub $%d, %%rsp\n", -curlocaloffset)
	}
	for idx, psym := range f.ParamSymbols {
		e.asm("movq %s, %d(%%rbp)\n", intParamLUT[idx], e.loffsets[psym])
	}
	for _, stmt := range f.Body {
		e.Stmt(stmt)
	}
	e.asm("leave\n")
	e.asm("ret\n")
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

func (e *emitter) Stmt(stmt parse.Node) {
	switch stmt := stmt.(type) {
	case *parse.If:
		e.If(stmt)
	case *parse.While:
		e.While(stmt)
	case *parse.DoWhile:
		e.DoWhile(stmt)
	case *parse.For:
		e.For(stmt)
	case *parse.Return:
		e.Return(stmt)
	case *parse.CompndStmt:
		e.CompndStmt(stmt)
	case *parse.ExprStmt:
		e.Expr(stmt.Expr)
	case *parse.Goto:
		e.asm("jmp %s\n", stmt.Label)
	case *parse.LabeledStmt:
		e.raw("%s:\n", stmt.AnonLabel)
		e.Stmt(stmt.Stmt)
	case *parse.Switch:
		e.Switch(stmt)
	case *parse.EmptyStmt:
		// pass
	case *parse.DeclList:
		// pass
	default:
		panic(stmt)
	}
}

func (e *emitter) Switch(sw *parse.Switch) {
	e.Expr(sw.Expr)
	for _, swc := range sw.Cases {
		e.asm("mov $%d, %%rcx\n", swc.V)
		e.asm("cmp %%rax, %%rcx\n")
		e.asm("je %s\n", swc.Label)
	}
	if sw.LDefault != "" {
		e.asm("jmp %s\n", sw.LDefault)
	} else {
		e.asm("jmp %s\n", sw.LAfter)
	}
	e.Stmt(sw.Stmt)
	e.raw("%s:\n", sw.LAfter)
}

func (e *emitter) While(w *parse.While) {
	e.raw("%s:\n", w.LStart)
	e.Expr(w.Cond)
	e.asm("test %%rax, %%rax\n")
	e.asm("jz %s\n", w.LEnd)
	e.Stmt(w.Body)
	e.asm("jmp %s\n", w.LStart)
	e.raw("%s:\n", w.LEnd)
}

func (e *emitter) DoWhile(d *parse.DoWhile) {
	e.raw("%s:\n", d.LStart)
	e.Stmt(d.Body)
	e.raw("%s:\n", d.LCond)
	e.Expr(d.Cond)
	e.asm("test %%rax, %%rax\n")
	e.asm("jz %s\n", d.LEnd)
	e.asm("jmp %s\n", d.LStart)
	e.raw("%s:\n", d.LEnd)
}

func (e *emitter) For(fr *parse.For) {
	if fr.Init != nil {
		e.Expr(fr.Init)
	}
	e.raw("%s:\n", fr.LStart)
	if fr.Cond != nil {
		e.Expr(fr.Cond)
	}
	e.asm("test %%rax, %%rax\n")
	e.asm("jz %s\n", fr.LEnd)
	e.Stmt(fr.Body)
	if fr.Step != nil {
		e.Expr(fr.Step)
	}
	e.asm("jmp %s\n", fr.LStart)
	e.raw("%s:\n", fr.LEnd)
}

func (e *emitter) CompndStmt(c *parse.CompndStmt) {
	for _, stmt := range c.Body {
		e.Stmt(stmt)
	}
}

func (e *emitter) If(i *parse.If) {
	e.Expr(i.Cond)
	e.asm("test %%rax, %%rax\n")
	e.asm("jz %s\n", i.LElse)
	e.Stmt(i.Stmt)
	e.raw("%s:\n", i.LElse)
	if i.Else != nil {
		e.Stmt(i.Else)
	}
}

func (e *emitter) Return(r *parse.Return) {
	e.Expr(r.Ret)
	e.asm("leave\n")
	e.asm("ret\n")
}

func (e *emitter) Expr(expr parse.Node) {
	switch expr := expr.(type) {
	case *parse.Ident:
		e.Ident(expr)
	case *parse.Call:
		e.Call(expr)
	case *parse.Constant:
		e.asm("movq $%v, %%rax\n", expr.Val)
	case *parse.Unop:
		e.Unop(expr)
	case *parse.Binop:
		e.emitBinop(expr)
	case *parse.Index:
		e.Index(expr)
	case *parse.Cast:
		e.emitCast(expr)
	case *parse.Selector:
		e.Selector(expr)
	case *parse.String:
		e.asm("leaq %s(%%rip), %%rax\n", expr.Label)
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

func (e *emitter) Selector(s *parse.Selector) {
	e.Expr(s.Operand)
	ty := s.Operand.GetType()
	offset := 0
	switch ty := ty.(type) {
	case *parse.CStruct:
		offset = getStructOffset(ty, s.Sel)
	default:
		panic("internal error")
	}
	if offset != 0 {
		e.asm("add $%d, %%rax\n", offset)
	}
	switch getSize(s.GetType()) {
	case 1:
		e.asm("movb (%%rax), %%al\n")
	case 4:
		e.asm("movl (%%rax), %%eax\n")
	case 8:
		e.asm("movq (%%rax), %%rax\n")
	default:
		panic("unimplemented")
	}
}

func (e *emitter) Ident(i *parse.Ident) {
	sym := i.Sym
	switch sym := sym.(type) {
	case *parse.LSymbol:
		offset := e.loffsets[sym]
		if parse.IsIntType(sym.Type) || parse.IsPtrType(sym.Type) {
			switch getSize(sym.Type) {
			case 1:
				e.asm("movb %d(%%rbp), %%al\n", offset)
			case 4:
				e.asm("movl %d(%%rbp), %%eax\n", offset)
			case 8:
				e.asm("movq %d(%%rbp), %%rax\n", offset)
			default:
				panic("unimplemented")
			}
		} else {
			e.asm("leaq %d(%%rbp), %%rax\n", offset)
		}
	case *parse.GSymbol:
		e.asm("leaq %s(%%rip), %%rax\n", sym.Label)
		if parse.IsIntType(sym.Type) || parse.IsPtrType(sym.Type) {
			switch getSize(sym.Type) {
			case 1:
				e.asm("movb (%%rax), %%al\n")
			case 4:
				e.asm("movl (%%rax), %%eax\n")
			case 8:
				e.asm("movq (%%rax), %%rax\n")
			default:
				panic("unimplemented")
			}
		}
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

func (e *emitter) Call(c *parse.Call) {
	intargs, memargs := classifyArgs(c.Args)
	sz := 0
	for i := len(memargs) - 1; i >= 0; i-- {
		arg := memargs[i]
		e.Expr(arg)
		e.asm("push %%rax\n")
		sz += 8
	}
	for i := len(intargs) - 1; i >= 0; i-- {
		arg := intargs[i]
		e.Expr(arg)
		e.asm("push %%rax\n")
	}
	for idx, _ := range intargs {
		e.asm("pop %s\n", intParamLUT[idx])
	}
	e.Expr(c.FuncLike)
	e.asm("call *%%rax\n")
	if sz != 0 {
		e.asm("add $%d, %%rsp\n", sz)
	}
}

func (e *emitter) emitCast(c *parse.Cast) {
	e.Expr(c.Operand)
	from := c.Operand.GetType()
	to := c.Type
	switch {
	case parse.IsPtrType(to):
		if parse.IsPtrType(from) {
			return
		}
		if parse.IsIntType(from) {
			switch getSize(to) {
			case 8:
				return
			case 4:
				// *NOTE* This zeros top half of rax.
				e.asm("mov %%eax, %%eax\n")
				return
			case 2:
				e.asm("movzwq %%ax, %%eax\n")
				return
			case 1:
				e.asm("movzbq %%al, %%eax\n")
				return
			}
		}
	case parse.IsIntType(to):
		if parse.IsPtrType(from) {
			// Free truncation
			return
		}
		if parse.IsIntType(from) {
			if getSize(to) <= getSize(from) {
				// Free truncation
				return
			}
			if parse.IsSignedIntType(from) {
				switch getSize(from) {
				case 4:
					e.asm("movsdq %%eax, %%rax\n")
				case 2:
					e.asm("movswq %%ax, %%rax\n")
				case 1:
					e.asm("movsbq %%al, %%rax\n")
				default:
					panic("internal error")
				}
			} else {
				switch getSize(to) {
				case 4:
					// *NOTE* This zeros top half of rax.
					e.asm("mov %%eax, %%eax\n")
					return
				case 2:
					e.asm("movzwq %%ax, %%eax\n")
					return
				case 1:
					e.asm("movzbq %%al, %%eax\n")
					return
				}
			}
			return
		}
	}
	panic("unimplemented cast")
}

func (e *emitter) emitBinop(b *parse.Binop) {
	if b.Op == '=' {
		e.Assign(b)
		return
	}
	e.Expr(b.L)
	e.asm("pushq %%rax\n")
	e.Expr(b.R)
	e.asm("movq %%rax, %%rcx\n")
	e.asm("popq %%rax\n")
	switch {
	case parse.IsIntType(b.Type):
		switch b.Op {
		case '+':
			e.asm("addq %%rcx, %%rax\n")
		case '-':
			e.asm("subq %%rcx, %%rax\n")
		case '*':
			e.asm("imul %%rcx, %%rax\n")
		case '|':
			e.asm("or %%rcx, %%rax\n")
		case '&':
			e.asm("and %%rcx, %%rax\n")
		case '^':
			e.asm("xor %%rcx, %%rax\n")
		case '/':
			e.asm("cqto\n")
			e.asm("idiv %%rcx\n")
		case '%':
			e.asm("idiv %%rcx\n")
			e.asm("mov %%rdx, %%rax\n")
		case cpp.SHL:
			e.asm("sal %%cl, %%rax\n")
		case cpp.SHR:
			e.asm("sar %%cl, %%rax\n")
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
			switch getSize(b.GetType()) {
			case 8:
				e.asm("cmp %%rcx, %%rax\n")
			case 4:
				e.asm("cmp %%ecx, %%eax\n")
			default:
				// There shouldn't be arith operations on anything else.
				panic("internal error")
			}
			e.asm("%s %s\n", opc, lset)
			e.asm("movq $0, %%rax\n")
			e.asm("jmp %s\n", lafter)
			e.asm("%s:\n", lset)
			e.asm("movq $1, %%rax\n")
			e.asm("%s:\n", lafter)
		default:
			panic("unimplemented " + b.Op.String())
		}
	default:
		panic(b)
	}
}

func (e *emitter) Unop(u *parse.Unop) {
	switch u.Op {
	case '&':
		switch operand := u.Operand.(type) {
		case *parse.Unop:
			if operand.Op != '*' {
				panic("internal error")
			}
			e.Expr(operand.Operand)
		case *parse.Ident:
			sym := operand.Sym
			switch sym := sym.(type) {
			case *parse.GSymbol:
				e.asm("leaq %s(%%rip), %%rax\n", sym.Label)
			case *parse.LSymbol:
				offset := e.loffsets[sym]
				e.asm("leaq %d(%%rbp), %%rax\n", offset)
			default:
				panic("internal error")
			}
		case *parse.Index:
			e.Expr(operand.Idx)
			sz := getSize(operand.GetType())
			e.asm("imul $%d, %%rax\n", sz)
			if sz != 1 {
				e.asm("imul $%d, %%rax\n", sz)
			}
			e.asm("push %%rax\n")
			e.Expr(operand.Arr)
			e.asm("pop %%rcx\n")
			e.asm("addq %%rcx, %%rax\n")
		default:
			panic("internal error")
		}
	case '!':
		e.Expr(u.Operand)
		e.asm("xor %%rcx, %%rcx\n")
		switch getSize(u.GetType()) {
		case 8:
			e.asm("test %%rax, %%rax\n")
		case 4:
			e.asm("test %%eax, %%eax\n")
		}
		e.asm("setnz %%cl\n")
		e.asm("mov %%rcx, %%rax\n")
	case '-':
		e.Expr(u.Operand)
		e.asm("neg %%rax\n")
	case '*':
		e.Expr(u.Operand)
		e.asm("movq (%%rax), %%rax\n")
	}
}

func (e *emitter) Index(idx *parse.Index) {
	e.Expr(idx.Idx)
	sz := getSize(idx.GetType())
	if sz != 1 {
		e.asm("imul $%d, %%rax\n", sz)
	}
	e.asm("push %%rax\n")
	e.Expr(idx.Arr)
	e.asm("pop %%rcx\n")
	e.asm("addq %%rcx, %%rax\n")
	switch getSize(idx.GetType()) {
	case 1:
		e.asm("movb (%%rax), %%al\n")
	case 4:
		e.asm("movl (%%rax), %%eax\n")
	case 8:
		e.asm("movq (%%rax), %%rax\n")
	}
}

func (e *emitter) Assign(b *parse.Binop) {
	e.Expr(b.R)
	switch l := b.L.(type) {
	case *parse.Index:
		e.asm("push %%rax\n")
		e.Expr(l.Idx)
		e.asm("imul $%d, %%rax\n", 4)
		e.asm("push %%rax\n")
		e.Expr(l.Arr)
		e.asm("pop %%rcx\n")
		e.asm("add %%rcx, %%rax\n")
		e.asm("pop %%rcx\n")
		e.asm("movq %%rcx,(%%rax)\n")
	case *parse.Selector:
		e.asm("push %%rax\n")
		e.Expr(l.Operand)
		ty := l.Operand.GetType()
		offset := 0
		switch ty := ty.(type) {
		case *parse.CStruct:
			offset = getStructOffset(ty, l.Sel)
		default:
			panic("internal error")
		}
		e.asm("add $%d,%%rax\n", offset)
		e.asm("pop %%rcx\n")
		switch getSize(ty) {
		case 1:
			e.asm("movb %%cl, (%%rax)\n")
		case 4:
			e.asm("movl %%ecx, (%%rax)\n")
		case 8:
			e.asm("movq %%rcx, (%%rax)\n")
		default:
			panic("unimplemented")
		}
	case *parse.Unop:
		if l.Op != '*' {
			panic("internal error")
		}
		e.asm("push %%rax\n")
		e.Expr(l.Operand)
		e.asm("pop %%rcx\n")
		e.asm("movq %%rcx, (%%rax)\n")
	case *parse.Ident:
		sym := l.Sym
		switch sym := sym.(type) {
		case *parse.GSymbol:
			e.asm("leaq %s(%%rip), %%rcx\n", sym.Label)
			switch getSize(sym.Type) {
			case 1:
				e.asm("movb %%al, (%%rcx)\n")
			case 4:
				e.asm("movl %%eax, (%%rcx)\n")
			case 8:
				e.asm("movq %%rax, (%%rcx)\n")
			default:
				panic("unimplemented")
			}
		case *parse.LSymbol:
			offset := e.loffsets[sym]
			switch getSize(sym.Type) {
			case 1:
				e.asm("movb %%al, %d(%%rbp)\n", offset)
			case 4:
				e.asm("movl %%eax, %d(%%rbp)\n", offset)
			case 8:
				e.asm("movq %%rax, %d(%%rbp)\n", offset)
			default:
				panic("unimplemented")
			}
		}
	default:
		panic(b.L)
	}
}
