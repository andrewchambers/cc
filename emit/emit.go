package emit

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
}

func (e *emitter) NextLabel() string {
	e.labelcounter += 1
	return fmt.Sprintf(".LL%d", e.labelcounter)
}

func Emit(toplevels []parse.Node, o io.Writer) error {
	e := &emitter{
		o: o,
	}

	e.emit(".data\n")
	for _, tl := range toplevels {
		e.emitStrings(tl)
	}
	for _, tl := range toplevels {
		switch tl := tl.(type) {
		case *parse.Function:
			e.emitFunction(tl)
		case *parse.DeclList:
			for idx, decl := range tl.Symbols {
				global, ok := decl.(*parse.GSymbol)
				if !ok {
					panic("internal error")
				}
				e.emitGlobal(global, tl.FoldedInits[idx])
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

func (e *emitter) emitStrings(n parse.Node) {
	s, ok := n.(*parse.String)
	if !ok {
		if n == nil {
			return //XXX this should not happen.
		}
		children := n.Children()
		for _, v := range children {
			e.emitStrings(v)
		}
		return
	}
	e.emit("%s:\n", s.Label)
	e.emit(".string %s\n", s.Val)
}

func (e *emitter) emitGlobal(g *parse.GSymbol, init *parse.FoldedConstant) {
	_, ok := g.Type.(*parse.FunctionType)
	if ok {
		return
	}
	e.emit(".data\n")
	e.emit(".global %s\n", g.Label)
	if init == nil {
		e.emit(".lcomm %s, %d\n", g.Label, g.Type.GetSize())
	} else {
		e.emit("%s:\n", g.Label)
		switch {
		case g.Type == parse.CInt:
			e.emit(".quad %v\n", init.Val)
		case parse.IsPtrType(g.Type):
			e.emit(".quad %s\n", init.Label)
		default:
		}
	}
}

var intParamLUT = [...]string{
	"%rdi", "%rsi", "%rdx", "%rcx", "r8", "r9",
}

func (e *emitter) emitFunction(f *parse.Function) {
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
		e.emitStmt(f, stmt)
	}
	e.emiti("leave\n")
	e.emiti("ret\n")
}

func (e *emitter) calcLocalOffsets(f *parse.Function) (int, map[*parse.LSymbol]int) {
	loffset := 0
	loffsets := make(map[*parse.LSymbol]int)
	addLSymbol := func(lsym *parse.LSymbol) {
		sz := lsym.Type.GetSize()
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

func (e *emitter) emitStmt(f *parse.Function, stmt parse.Node) {
	switch stmt := stmt.(type) {
	case *parse.If:
		e.emitIf(f, stmt)
	case *parse.While:
		e.emitWhile(f, stmt)
	case *parse.DoWhile:
		e.emitDoWhile(f, stmt)
	case *parse.For:
		e.emitFor(f, stmt)
	case *parse.Return:
		e.emitReturn(f, stmt)
	case *parse.CompndStmt:
		e.emitCompndStmt(f, stmt)
	case *parse.ExprStmt:
		e.emitExpr(f, stmt.Expr)
	case *parse.Goto:
		e.emiti("jmp %s\n", stmt.Label)
	case *parse.LabeledStmt:
		e.emit("%s:\n", stmt.AnonLabel)
		e.emitStmt(f, stmt.Stmt)
	case *parse.Switch:
		e.emitSwitch(f, stmt)
	case *parse.EmptyStmt:
		// pass
	case *parse.DeclList:
		// pass
	default:
		panic(stmt)
	}
}

func (e *emitter) emitSwitch(f *parse.Function, sw *parse.Switch) {
	e.emitExpr(f, sw.Expr)
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
	e.emitStmt(f, sw.Stmt)
	e.emit("%s:\n", sw.LAfter)
}

func (e *emitter) emitWhile(f *parse.Function, w *parse.While) {
	e.emit("%s:\n", w.LStart)
	e.emitExpr(f, w.Cond)
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", w.LEnd)
	e.emitStmt(f, w.Body)
	e.emiti("jmp %s\n", w.LStart)
	e.emit("%s:\n", w.LEnd)
}

func (e *emitter) emitDoWhile(f *parse.Function, d *parse.DoWhile) {
	e.emit("%s:\n", d.LStart)
	e.emitStmt(f, d.Body)
	e.emit("%s:\n", d.LCond)
	e.emitExpr(f, d.Cond)
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", d.LEnd)
	e.emiti("jmp %s\n", d.LStart)
	e.emit("%s:\n", d.LEnd)
}

func (e *emitter) emitFor(f *parse.Function, fr *parse.For) {
	if fr.Init != nil {
		e.emitExpr(f, fr.Init)
	}
	e.emit("%s:\n", fr.LStart)
	if fr.Cond != nil {
		e.emitExpr(f, fr.Cond)
	}
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", fr.LEnd)
	e.emitStmt(f, fr.Body)
	if fr.Step != nil {
		e.emitExpr(f, fr.Step)
	}
	e.emiti("jmp %s\n", fr.LStart)
	e.emit("%s:\n", fr.LEnd)
}

func (e *emitter) emitCompndStmt(f *parse.Function, c *parse.CompndStmt) {
	for _, stmt := range c.Body {
		e.emitStmt(f, stmt)
	}
}

func (e *emitter) emitIf(f *parse.Function, i *parse.If) {
	e.emitExpr(f, i.Cond)
	e.emiti("test %%rax, %%rax\n")
	e.emiti("jz %s\n", i.LElse)
	e.emitStmt(f, i.Stmt)
	e.emit("%s:\n", i.LElse)
	if i.Else != nil {
		e.emitStmt(f, i.Else)
	}
}

func (e *emitter) emitReturn(f *parse.Function, r *parse.Return) {
	e.emitExpr(f, r.Ret)
	e.emiti("leave\n")
	e.emiti("ret\n")
}

func (e *emitter) emitExpr(f *parse.Function, expr parse.Node) {
	switch expr := expr.(type) {
	case *parse.Ident:
		e.emitIdent(f, expr)
	case *parse.Call:
		e.emitCall(f, expr)
	case *parse.Constant:
		e.emiti("movq $%v, %%rax\n", expr.Val)
	case *parse.Unop:
		e.emitUnop(f, expr)
	case *parse.Binop:
		e.emitBinop(f, expr)
	case *parse.Index:
		e.emitIndex(f, expr)
	case *parse.Cast:
		e.emitCast(f, expr)
	default:
		panic(expr)
	}
}

func (e *emitter) emitIdent(f *parse.Function, i *parse.Ident) {
	sym := i.Sym
	switch sym := sym.(type) {
	case *parse.LSymbol:
		offset := e.loffsets[sym]
		if parse.IsIntType(sym.Type) || parse.IsPtrType(sym.Type) {
			switch sym.Type.GetSize() {
			case 1:
				e.emiti("movb %d(%%rbp), %%al\n", offset)
			case 4:
				e.emiti("movl %d(%%rbp), %%eax\n", offset)
			case 8:
				e.emiti("movq %d(%%rbp), %%rax\n", offset)
			default:
				panic("unimplemented")
			}
		} else {
			e.emiti("leaq %d(%%rbp), %%rax\n", offset)
		}
	case *parse.GSymbol:
		e.emiti("leaq %s(%%rip), %%rax\n", sym.Label)
		if parse.IsIntType(sym.Type) || parse.IsPtrType(sym.Type) {
			switch sym.Type.GetSize() {
			case 1:
				e.emiti("movb (%%rax), %%al\n")
			case 4:
				e.emiti("movl (%%rax), %%eax\n")
			case 8:
				e.emiti("movq (%%rax), %%rax\n")
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

func (e *emitter) emitCall(f *parse.Function, c *parse.Call) {
	intargs, memargs := classifyArgs(c.Args)
	sz := 0
	for i := len(memargs) - 1; i >= 0; i-- {
		arg := memargs[i]
		e.emitExpr(f, arg)
		e.emiti("push %%rax\n")
		sz += 8
	}
	for i := len(intargs) - 1; i >= 0; i-- {
		arg := intargs[i]
		e.emitExpr(f, arg)
		e.emiti("push %%rax\n")
	}
	for idx, _ := range intargs {
		e.emiti("pop %s\n", intParamLUT[idx])
	}
	e.emitExpr(f, c.FuncLike)
	e.emiti("call *%%rax\n")
	if sz != 0 {
		e.emiti("add $%d, %%rsp\n", sz)
	}
}

func (e *emitter) emitCast(f *parse.Function, c *parse.Cast) {
	//
}

func (e *emitter) emitBinop(f *parse.Function, b *parse.Binop) {
	if b.Op == '=' {
		e.emitAssign(f, b)
		return
	}
	e.emitExpr(f, b.L)
	e.emiti("pushq %%rax\n")
	e.emitExpr(f, b.R)
	e.emiti("popq %%rcx\n")
	switch {
	case parse.IsIntType(b.Type):
		switch b.Op {
		case '+':
			e.emiti("addq %%rax, %%rcx\n")
			e.emiti("movq %%rcx, %%rax\n")
		case '-':
			e.emiti("subq %%rax, %%rcx\n")
			e.emiti("movq %%rcx, %%rax\n")
		case '*':
			e.emiti("imul %%rax, %%rcx\n")
			e.emiti("movq %%rcx, %%rax\n")
		case cpp.EQL, '>', '<':
			leq := e.NextLabel()
			lafter := e.NextLabel()
			opc := ""
			switch b.Op {
			case cpp.EQL:
				opc = "je"
			case '<':
				opc = "jl"
			case '>':
				opc = "jg"
			default:
				panic("internal error")
			}
			e.emiti("cmp %%rax, %%rcx\n")
			e.emiti("%s %s\n", opc, leq)
			e.emiti("movq $0, %%rax\n")
			e.emiti("jmp %s\n", lafter)
			e.emiti("%s:\n", leq)
			e.emiti("movq $1, %%rax\n")
			e.emiti("%s:\n", lafter)
		default:
			panic("unimplemented")
		}
	default:
		panic(b.Type)
	}
}

func (e *emitter) emitUnop(f *parse.Function, u *parse.Unop) {
	switch u.Op {
	case '&':
		switch operand := u.Operand.(type) {
		case *parse.Unop:
			if operand.Op != '*' {
				panic("internal error")
			}
			e.emitExpr(f, operand.Operand)
		case *parse.Ident:
			sym := operand.Sym
			switch sym := sym.(type) {
			case *parse.GSymbol:
				e.emiti("leaq %s(%%rip), %%rax\n", sym.Label)
			}
		case *parse.Index:
			e.emitExpr(f, operand.Idx)
			sz := operand.GetType().GetSize()
			e.emiti("imul $%d, %%rax\n", sz)
			if sz != 1 {
				e.emiti("imul $%d, %%rax\n", sz)
			}
			e.emiti("push %%rax\n")
			e.emitExpr(f, operand.Arr)
			e.emiti("pop %%rcx\n")
			e.emiti("addq %%rcx, %%rax\n")
		default:
			panic("internal error")
		}
	case '-':
		e.emitExpr(f, u.Operand)
		e.emiti("neg %%rax\n")
	case '*':
		e.emitExpr(f, u.Operand)
		e.emiti("movq (%%rax), %%rax\n")
	}
}

func (e *emitter) emitIndex(f *parse.Function, idx *parse.Index) {
	e.emitExpr(f, idx.Idx)
	sz := idx.GetType().GetSize()
	if sz != 1 {
		e.emiti("imul $%d, %%rax\n", sz)
	}
	e.emiti("push %%rax\n")
	e.emitExpr(f, idx.Arr)
	e.emiti("pop %%rcx\n")
	e.emiti("addq %%rcx, %%rax\n")
	switch idx.GetType().GetSize() {
	case 1:
		e.emiti("movb (%%rax), %%al\n")
	case 4:
		e.emiti("movl (%%rax), %%eax\n")
	case 8:
		e.emiti("movq (%%rax), %%rax\n")
	}
}

func (e *emitter) emitAssign(f *parse.Function, b *parse.Binop) {
	e.emitExpr(f, b.R)
	switch l := b.L.(type) {
	case *parse.Index:
		e.emiti("push %%rax\n")
		e.emitExpr(f, l.Idx)
		e.emiti("imul $%d, %%rax\n", 4)
		e.emiti("push %%rax\n")
		e.emitExpr(f, l.Arr)
		e.emiti("pop %%rcx\n")
		e.emiti("add %%rcx, %%rax\n")
		e.emiti("pop %%rcx\n")
		e.emiti("movq %%rcx,(%%rax)\n")
	case *parse.Unop:
		if l.Op != '*' {
			panic("internal error")
		}
		e.emiti("push %%rax\n")
		e.emitExpr(f, l.Operand)
		e.emiti("pop %%rcx\n")
		e.emiti("movq %%rcx, (%%rax)\n")
	case *parse.Ident:
		sym := l.Sym
		switch sym := sym.(type) {
		case *parse.GSymbol:
			e.emiti("leaq %s(%%rip), %%rcx\n", sym.Label)
			switch sym.Type.GetSize() {
			case 1:
				e.emiti("movb %%al, (%%rcx)\n")
			case 4:
				e.emiti("movl %%eax, (%%rcx)\n")
			case 8:
				e.emiti("movq %%rax, (%%rcx)\n")
			default:
				panic("unimplemented")
			}
		case *parse.LSymbol:
			offset := e.loffsets[sym]
			switch sym.Type.GetSize() {
			case 1:
				e.emiti("movb %%ax, %d(%%rbp)\n", offset)
			case 4:
				e.emiti("movl %%eax, %d(%%rbp)\n", offset)
			case 8:
				e.emiti("movq %%rax, %d(%%rbp)\n", offset)
			default:
				panic("unimplemented")
			}
		}
	}
}
