package cpp

import (
	"container/list"
	"fmt"
	"io"
)

// XXX rewrite this to use a stack of lexers
// The way it handles EOF is currently broken.

type Preprocessor struct {
	is IncludeSearcher
	//List of all pushed back tokens
	tl *tokenList
	//Map of defined macros
	objMacros map[string]*objMacro
	//Map of defined FUNC macros
	funcMacros map[string]*funcMacro
	//Where the tokens are to be sent
	out chan *Token
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

func New(is IncludeSearcher) *Preprocessor {
	ret := new(Preprocessor)
	ret.is = is
	ret.tl = newTokenList()
	ret.objMacros = make(map[string]*objMacro)
	ret.funcMacros = make(map[string]*funcMacro)
	ret.conditionalStack = list.New()
	return ret
}

func emptyTokChan(t chan *Token) {
	for _ = range t {
		//PASS
	}
}

func (pp *Preprocessor) nextToken(in chan *Token) *Token {
	if pp.tl.isEmpty() {
		t := <-in
		if t.Kind == EOF {
			return nil
		}
		return t
	}
	return pp.tl.popFront()
}

func (pp *Preprocessor) nextTokenExpand(in chan *Token) *Token {
	t := pp.nextToken(in)
	if t == nil {
		return nil
	}
	if t.hs.contains(t.Val) {
		return t
	}
	macro, ok := pp.objMacros[t.Val]
	if ok {
		replacementTokens := macro.tokens.copy()
		replacementTokens.addToHideSets(t)
		replacementTokens.setPositions(t.Pos)
		pp.ungetTokens(replacementTokens)
		return pp.nextTokenExpand(in)
	}
	fmacro, ok := pp.funcMacros[t.Val]
	if ok {
		opening := pp.nextToken(in)
		if opening != nil && opening.Kind == LPAREN {
			args, rparen, err := pp.readMacroInvokeArguments(in)
			if len(args) != fmacro.nargs {
				pp.cppError(fmt.Sprintf(
					"macro %s invoked with %d arguments but %d were expected",
					t.Val, len(args), fmacro.nargs), t.Pos)
			}
			if err != nil {
				pp.cppError(err.Error(), t.Pos)
			}
			hs := t.hs.intersection(rparen.hs)
			hs = hs.add(t.Val)
			pp.subst(fmacro, t.Pos, args, hs)
			return pp.nextTokenExpand(in)
		}
	}
	return t
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
func (pp *Preprocessor) readMacroInvokeArguments(in chan *Token) ([]*tokenList, *Token, error) {
	parenDepth := 1
	argIdx := 0
	ret := make([]*tokenList, 0, 16)
	ret = append(ret, newTokenList())
	for {
		t := pp.nextToken(in)
		if t == nil {
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

func (pp *Preprocessor) cppError(e string, pos FilePos) {
	emsg := fmt.Sprintf("Preprocessor error %s at %s", e, pos)
	pp.out <- &Token{Kind: ERROR, Val: emsg, Pos: pos}
	//recover exits cleanly
	panic(&breakout{})
}

//The preprocessor can only be run once. Create a new one to reuse.
func (pp *Preprocessor) Preprocess(in chan *Token) chan *Token {
	out := make(chan *Token)
	pp.out = out
	go pp.preprocess(in)
	return out
}

func (pp *Preprocessor) preprocess(in chan *Token) {
	defer func() {
		//XXX is this correct way to retrigger non breakout?
		if e := recover(); e != nil {
			_ = e.(*breakout) // Will re-panic if not a parse error.
			close(pp.out)
		}
	}()
	pp.preprocess2(in)
	close(pp.out)
}

func (pp *Preprocessor) preprocess2(in chan *Token) {
	//We have to run the lexer dry or it is a leak.
	defer emptyTokChan(in)
	for {
		tok := pp.nextTokenExpand(in)
		if tok == nil {
			break
		}
		switch tok.Kind {
		case ERROR:
			pp.out <- tok
			panic(&breakout{})
		case DIRECTIVE:
			pp.handleDirective(tok, in)
		default:
			pp.out <- tok
		}
	}
}

func (pp *Preprocessor) handleIf(pos FilePos, in chan *Token) {
	pp.pushCondContext()
	//Pretend it fails...
	pp.skipTillEndif(pos, in)
}

func (pp *Preprocessor) handleIfDef(pos FilePos, in chan *Token) {
	pp.pushCondContext()
	//Pretend it fails...
	pp.skipTillEndif(pos, in)
}

func (pp *Preprocessor) handleEndif(pos FilePos, in chan *Token) {
	if pp.condDepth() <= 0 {
		pp.cppError("stray #endif", pos)
	}
	pp.popCondContext()
	endTok := pp.nextToken(in)
	if endTok.Kind != END_DIRECTIVE {
		pp.cppError("unexpected token after #endif", endTok.Pos)
	}
}

//XXX untested
func (pp *Preprocessor) skipTillEndif(pos FilePos, in chan *Token) {
	depth := 1
	for {
		//Dont care about expands since we are skipping.
		t := pp.nextToken(in)
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

func (pp *Preprocessor) handleDirective(dirTok *Token, in chan *Token) {
	if dirTok.Kind != DIRECTIVE {
		pp.cppError(fmt.Sprintf("internal error %s", dirTok), dirTok.Pos)
	}
	switch dirTok.Val {
	case "if":
		pp.handleIf(dirTok.Pos, in)
	case "ifdef":
		pp.handleIfDef(dirTok.Pos, in)
	case "endif":
		pp.handleEndif(dirTok.Pos, in)
	//case "ifndef":
	//case "elif":
	//case "else":
	case "undef":
		pp.handleUndefine(in)
	case "define":
		pp.handleDefine(in)
	case "include":
		pp.handleInclude(in)
	case "error":
		pp.handleError(in)
	case "warning":
		pp.handleWarning(in)
	default:
		pp.cppError(fmt.Sprintf("unknown directive error %s", dirTok), dirTok.Pos)
	}
}

func (pp *Preprocessor) handleError(in chan *Token) {
	tok := pp.nextToken(in)
	if tok.Kind != STRING {
		pp.cppError("error string %s", tok.Pos)
	}
	pp.cppError(tok.String(), tok.Pos)
}

func (pp *Preprocessor) handleWarning(in chan *Token) {
	//XXX
	pp.handleError(in)
}

func (pp *Preprocessor) handleInclude(in chan *Token) {
	tok := pp.nextToken(in)
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
	if err != nil {
		pp.cppError(fmt.Sprintf("error during include %s", err), tok.Pos)
	}
	pp.preprocess2(Lex(headerName, rdr))
	tok = pp.nextToken(in)
	if tok.Kind != END_DIRECTIVE {
		pp.cppError("Expected newline after include %s", tok.Pos)
	}
}

func (pp *Preprocessor) handleUndefine(in chan *Token) {
	ident := pp.nextToken(in)
	if ident.Kind != IDENT {
		pp.cppError("#undefine expected an ident", ident.Pos)
	}
	if !pp.isDefined(ident.Val) {
		pp.cppError(fmt.Sprintf("cannot undefine %s, not defined", ident.Val), ident.Pos)
	}
	delete(pp.objMacros, ident.Val)
	delete(pp.funcMacros, ident.Val)
	end := pp.nextToken(in)
	if end.Kind != END_DIRECTIVE {
		pp.cppError("expected end of directive", end.Pos)
	}
}

func (pp *Preprocessor) handleDefine(in chan *Token) {
	ident := pp.nextToken(in)
	//XXX should also support keywords and maybe other things
	if ident.Kind != IDENT {
		pp.cppError("#define expected an ident", ident.Pos)
	}
	t := pp.nextToken(in)
	if t.Kind == FUNCLIKE_DEFINE {
		pp.handleFuncLikeDefine(ident, in)
	} else {
		pp.ungetToken(t)
		pp.handleObjDefine(ident, in)
	}

}

func (pp *Preprocessor) isDefined(s string) bool {
	_, ok1 := pp.funcMacros[s]
	_, ok2 := pp.objMacros[s]
	return ok1 || ok2
}

func (pp *Preprocessor) handleFuncLikeDefine(ident *Token, in chan *Token) {
	//First read the arguments.
	paren := pp.nextToken(in)
	if paren.Kind != LPAREN {
		panic("Bug, func like define without opening LPAREN")
	}

	if pp.isDefined(ident.Val) {
		pp.cppError("macro redefinition "+ident.Val, ident.Pos)
	}

	args := newTokenList()
	tokens := newTokenList()

	for {
		t := pp.nextToken(in)
		if t.Kind == RPAREN {
			break
		}
		if t.Kind != IDENT {
			pp.cppError("Expected macro argument", t.Pos)
		}
		args.append(t)
		t2 := pp.nextToken(in)
		if t2.Kind == COMMA {
			continue
		} else if t2.Kind == RPAREN {
			break
		} else {
			pp.cppError("Error in macro definition expected , or )", t2.Pos)
		}
	}

	for {
		t := pp.nextToken(in)
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

func (pp *Preprocessor) handleObjDefine(ident *Token, in chan *Token) {
	if pp.isDefined(ident.Val) {
		pp.cppError("macro redefinition "+ident.Val, ident.Pos)
	}
	tl := newTokenList()
	for {
		t := pp.nextToken(in)
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