package cpp

import (
	"container/list"
	"fmt"
	"io"
)

type Preprocessor struct {
	lxidx  int
	lexers [1024]*Lexer

	is IncludeSearcher
	//List of all pushed back tokens
	tl *tokenList
	//Map of defined macros
	objMacros map[string]*objMacro
	//Map of defined FUNC macros
	funcMacros map[string]*funcMacro

	//Stack of condContext about #ifdefs blocks
	conditionalStack *list.List
}

type condContext struct {
	hasSucceeded bool
}

func (pp *Preprocessor) pushCondContext() {
	pp.conditionalStack.PushBack(&condContext{false})
}

func (pp *Preprocessor) popCondContext() {
	if pp.condDepth() == 0 {
		panic("internal bug")
	}
	pp.conditionalStack.Remove(pp.conditionalStack.Back())
}

func (pp *Preprocessor) markCondContextSucceeded() {
	pp.conditionalStack.Back().Value.(*condContext).hasSucceeded = true
}

func (pp *Preprocessor) condDepth() int {
	return pp.conditionalStack.Len()
}

func New(l *Lexer, is IncludeSearcher) *Preprocessor {
	ret := new(Preprocessor)
	ret.lexers[0] = l
	ret.is = is
	ret.tl = newTokenList()
	ret.objMacros = make(map[string]*objMacro)
	ret.funcMacros = make(map[string]*funcMacro)
	ret.conditionalStack = list.New()
	return ret
}

type cppbreakout struct {
	t   *Token
	err error
}

func (pp *Preprocessor) nextNoExpand() *Token {
	if pp.tl.isEmpty() {
		for {
			t, err := pp.lexers[pp.lxidx].Next()
			if err != nil {
				panic(&cppbreakout{t, err})
			}
			if t.Kind == EOF {
				if pp.lxidx == 0 {
					return t
				}
				pp.lxidx -= 1
				continue
			}
			return t
		}
	}
	return pp.tl.popFront()
}

func (pp *Preprocessor) cppError(e string, pos FilePos) {
	err := fmt.Errorf("%s at %s", e, pos)
	panic(&cppbreakout{
		t:   &Token{},
		err: err,
	})
}

func (pp *Preprocessor) Next() (t *Token, err error) {

	defer func() {
		if e := recover(); e != nil {
			var b *cppbreakout
			b = e.(*cppbreakout)
			t = b.t
			err = b.err
		}
	}()

	t = pp.nextNoExpand()

	for t.Kind == DIRECTIVE {
		pp.handleDirective(t)
		t = pp.nextNoExpand()
	}

	if t.hs.contains(t.Val) {
		return t, nil
	}
	macro, ok := pp.objMacros[t.Val]
	if ok {
		replacementTokens := macro.tokens.copy()
		replacementTokens.addToHideSets(t)
		replacementTokens.setPositions(t.Pos)
		pp.ungetTokens(replacementTokens)
		return pp.Next()
	}
	fmacro, ok := pp.funcMacros[t.Val]
	if ok {
		opening := pp.nextNoExpand()
		if opening.Kind == LPAREN {
			args, rparen, err := pp.readMacroInvokeArguments()
			if len(args) != fmacro.nargs {
				return &Token{}, fmt.Errorf("macro %s invoked with %d arguments but %d were expected at %s", t.Val, len(args), fmacro.nargs, t.Pos)
			}
			if err != nil {
				return &Token{}, err
			}
			hs := t.hs.intersection(rparen.hs)
			hs = hs.add(t.Val)
			pp.subst(fmacro, t.Pos, args, hs)
			return pp.Next()
		}
	}
	return t, nil
}

func (pp *Preprocessor) subst(macro *funcMacro, invokePos FilePos, args []*tokenList, hs *hideset) {
	expandedTokens := newTokenList()
	for e := macro.tokens.front(); e != nil; e = e.Next() {
		t := e.Value.(*Token)
		idx, tIsArg := macro.isArg(t)
		if tIsArg {
			expandedTokens.appendList(args[idx])
		} else {
			tcpy := t.copy()
			tcpy.Pos = invokePos
			expandedTokens.append(tcpy)
		}
	}
	expandedTokens.setHideSets(hs)
	pp.ungetTokens(expandedTokens)
}

//Read the tokens that are part of a macro invocation, not including the first paren.
//But including the last paren. Handles nested parens.
//returns a slice of token lists and the closing paren.
//Each token list in the returned value represents a read macro param.
//e.g. FOO(BAR,(A,B),C)  -> { <BAR> , <(A,B)> , <C> } , )
//Where FOO( has already been consumed.
func (pp *Preprocessor) readMacroInvokeArguments() ([]*tokenList, *Token, error) {
	parenDepth := 1
	argIdx := 0
	ret := make([]*tokenList, 0, 16)
	ret = append(ret, newTokenList())
	for {
		t := pp.nextNoExpand()
		if t.Kind == EOF {
			return nil, nil, fmt.Errorf("EOF while reading macro arguments")
		}
		switch t.Kind {
		case LPAREN:
			parenDepth += 1
			if parenDepth != 1 {
				ret[argIdx].append(t)
			}
		case RPAREN:
			parenDepth -= 1
			if parenDepth == 0 {
				return ret, t, nil
			} else {
				ret[argIdx].append(t)
			}
		case COMMA:
			if parenDepth == 1 {
				//nextArg
				argIdx += 1
				ret = append(ret, newTokenList())
			} else {
				ret[argIdx].append(t)
			}
		default:
			ret[argIdx].append(t)
		}
	}
}

func (pp *Preprocessor) ungetTokens(tl *tokenList) {
	pp.tl.prependList(tl)
}

func (pp *Preprocessor) ungetToken(t *Token) {
	pp.tl.prepend(t)
}

func (pp *Preprocessor) handleIf(pos FilePos) {
	pp.pushCondContext()
	//Pretend it fails...
	pp.skipTillEndif(pos)
}

func (pp *Preprocessor) handleIfDef(pos FilePos) {
	pp.pushCondContext()
	//Pretend it fails...
	pp.skipTillEndif(pos)
}

func (pp *Preprocessor) handleEndif(pos FilePos) {
	if pp.condDepth() <= 0 {
		pp.cppError("stray #endif", pos)
	}
	pp.popCondContext()
	endTok := pp.nextNoExpand()
	if endTok.Kind != END_DIRECTIVE {
		pp.cppError("unexpected token after #endif", endTok.Pos)
	}
}

//XXX untested
func (pp *Preprocessor) skipTillEndif(pos FilePos) {
	depth := 1
	for {
		//Dont care about expands since we are skipping.
		t := pp.nextNoExpand()
		if t == nil {
			pp.cppError("unclosed preprocessor conditional", pos)
		}

		if t.Kind == DIRECTIVE && (t.Val == "if" || t.Val == "ifdef" || t.Val == "ifndef") {
			depth += 1
			continue
		}

		if t.Kind == DIRECTIVE && t.Val == "endif" {
			depth -= 1
		}

		if depth == 0 {
			break
		}
	}
}

func (pp *Preprocessor) handleDirective(dirTok *Token) {
	if dirTok.Kind != DIRECTIVE {
		pp.cppError(fmt.Sprintf("internal error %s", dirTok), dirTok.Pos)
	}
	switch dirTok.Val {
	case "if":
		pp.handleIf(dirTok.Pos)
	case "ifdef":
		pp.handleIfDef(dirTok.Pos)
	case "endif":
		pp.handleEndif(dirTok.Pos)
	//case "ifndef":
	//case "elif":
	//case "else":
	case "undef":
		pp.handleUndefine()
	case "define":
		pp.handleDefine()
	case "include":
		pp.handleInclude()
	case "error":
		pp.handleError()
	case "warning":
		pp.handleWarning()
	default:
		pp.cppError(fmt.Sprintf("unknown directive error %s", dirTok), dirTok.Pos)
	}
}

func (pp *Preprocessor) handleError() {
	tok := pp.nextNoExpand()
	if tok.Kind != STRING {
		pp.cppError("error string %s", tok.Pos)
	}
	pp.cppError(tok.String(), tok.Pos)
}

func (pp *Preprocessor) handleWarning() {
	//XXX
	pp.handleError()
}

func (pp *Preprocessor) handleInclude() {
	tok := pp.nextNoExpand()
	if tok.Kind != HEADER {
		pp.cppError("expected a header at %s", tok.Pos)
	}
	headerStr := tok.Val
	path := headerStr[1 : len(headerStr)-1]
	var headerName string
	var rdr io.Reader
	var err error
	switch headerStr[0] {
	case '<':
		headerName, rdr, err = pp.is.IncludeAngled(tok.Pos.File, path)
	case '"':
		headerName, rdr, err = pp.is.IncludeQuote(tok.Pos.File, path)
	default:
		pp.cppError("internal error %s", tok.Pos)
	}
	tok = pp.nextNoExpand()
	if tok.Kind != END_DIRECTIVE {
		pp.cppError("Expected newline after include", tok.Pos)
	}
	if err != nil {
		pp.cppError(fmt.Sprintf("error during include %s", err), tok.Pos)
	}
	pp.lxidx += 1
	pp.lexers[pp.lxidx] = Lex(headerName, rdr)
}

func (pp *Preprocessor) handleUndefine() {
	ident := pp.nextNoExpand()
	if ident.Kind != IDENT {
		pp.cppError("#undefine expected an ident", ident.Pos)
	}
	if !pp.isDefined(ident.Val) {
		pp.cppError(fmt.Sprintf("cannot undefine %s, not defined", ident.Val), ident.Pos)
	}
	delete(pp.objMacros, ident.Val)
	delete(pp.funcMacros, ident.Val)
	end := pp.nextNoExpand()
	if end.Kind != END_DIRECTIVE {
		pp.cppError("expected end of directive", end.Pos)
	}
}

func (pp *Preprocessor) handleDefine() {
	ident := pp.nextNoExpand()
	//XXX should also support keywords and maybe other things
	if ident.Kind != IDENT {
		pp.cppError("#define expected an ident", ident.Pos)
	}
	t := pp.nextNoExpand()
	if t.Kind == FUNCLIKE_DEFINE {
		pp.handleFuncLikeDefine(ident)
	} else {
		pp.ungetToken(t)
		pp.handleObjDefine(ident)
	}

}

func (pp *Preprocessor) isDefined(s string) bool {
	_, ok1 := pp.funcMacros[s]
	_, ok2 := pp.objMacros[s]
	return ok1 || ok2
}

func (pp *Preprocessor) handleFuncLikeDefine(ident *Token) {
	//First read the arguments.
	paren := pp.nextNoExpand()
	if paren.Kind != LPAREN {
		panic("Bug, func like define without opening LPAREN")
	}

	if pp.isDefined(ident.Val) {
		pp.cppError("macro redefinition "+ident.Val, ident.Pos)
	}

	args := newTokenList()
	tokens := newTokenList()

	for {
		t := pp.nextNoExpand()
		if t.Kind == RPAREN {
			break
		}
		if t.Kind != IDENT {
			pp.cppError("Expected macro argument", t.Pos)
		}
		args.append(t)
		t2 := pp.nextNoExpand()
		if t2.Kind == COMMA {
			continue
		} else if t2.Kind == RPAREN {
			break
		} else {
			pp.cppError("Error in macro definition expected , or )", t2.Pos)
		}
	}

	for {
		t := pp.nextNoExpand()
		if t.Kind == END_DIRECTIVE {
			break
		}
		tokens.append(t)
	}

	macro, err := newFuncMacro(args, tokens)
	if err != nil {
		pp.cppError("Error in macro definition "+err.Error(), ident.Pos)
	}
	pp.funcMacros[ident.Val] = macro
}

func (pp *Preprocessor) handleObjDefine(ident *Token) {
	if pp.isDefined(ident.Val) {
		pp.cppError("macro redefinition "+ident.Val, ident.Pos)
	}
	tl := newTokenList()
	for {
		t := pp.nextNoExpand()
		if t == nil {
			panic("Bug, EOF before END_DIRECTIVE in define at" + t.String())
		}
		if t.Kind == END_DIRECTIVE {
			break
		}
		tl.append(t)
	}
	m := newObjMacro(tl)
	pp.objMacros[ident.Val] = m
}
