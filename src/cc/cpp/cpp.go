package cpp

import (
	"fmt"
	"io"
)

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
}

func New(is IncludeSearcher) *Preprocessor {
	ret := new(Preprocessor)
	ret.is = is
	ret.tl = newTokenList()
	ret.objMacros = make(map[string]*objMacro)
	ret.funcMacros = make(map[string]*funcMacro)
	return ret
}

func (pp *Preprocessor) nextToken(in chan *Token) *Token {
	if pp.tl.isEmpty() {
		return <-in
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
		//fmt.Println("expanding ", t.Val, "to", replacementTokens)
		pp.ungetTokens(replacementTokens)
		return pp.nextTokenExpand(in)
	}

	_, ok = pp.funcMacros[t.Val]
	if ok {
		opening := pp.nextToken(in)
		if opening.Kind == LPAREN {
			_, _ = pp.readMacroInvokeArguments(in)
			panic("args...")
		} else {
			panic("macro() with no opening paren")
		}
	}

	return t
}

//Read the tokens that are part of a macro invocation, not including the first paren.
//But including the last paren. Handles nested parens.
//returns a slice of token lists. Each token list represents a read macro param.
//e.g. FOO(BAR,(A,B),C)  -> { <BAR> , <(A,B)> , <C> }
// Where FOO( has already been consumed.
func (pp *Preprocessor) readMacroInvokeArguments(in chan *Token) ([]*tokenList, error) {
	parenDepth := 1
	argIdx := 0
	ret := make([]*tokenList, 0, 16)
	ret = append(ret, newTokenList())
	for {
		t := pp.nextToken(in)
		if t == nil {
			return nil, fmt.Errorf("EOF while reading macro arguments")
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
				break
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
	return ret, nil
}

func (pp *Preprocessor) ungetTokens(tl *tokenList) {
	pp.tl.appendList(tl)
}

func (pp *Preprocessor) ungetToken(t *Token) {
	pp.tl.append(t)
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

func emptyTokChan(t chan *Token) {
	for _ = range t {
		//PASS
	}
}

func (pp *Preprocessor) handleDirective(dirTok *Token, in chan *Token) {
	if dirTok.Kind != DIRECTIVE {
		pp.cppError(fmt.Sprintf("internal error %s", dirTok), dirTok.Pos)
	}
	switch dirTok.Val {
	//case "if":
	//case "ifdef":
	//case "ifndef":
	//case "elif":
	//case "else":
	//case "endif":
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

func (pp *Preprocessor) handleFuncLikeDefine(ident *Token, in chan *Token) {
	//First read the arguments.
	paren := pp.nextToken(in)
	if paren.Kind != LPAREN {
		panic("Bug, func like define without opening LPAREN")
	}
	for {
		t := pp.nextToken(in)
		if t.Kind == RPAREN {
			break
		}
		if t.Kind != IDENT {
			pp.cppError("Expected macro argument", t.Pos)
		}
		t2 := pp.nextToken(in)
		if t2.Kind == COMMA {

		} else if t2.Kind == RPAREN {
			break
		} else {
		}
	}

	for {
		t := pp.nextToken(in)
		if t.Kind == END_DIRECTIVE {
			break
		}
	}
}

func (pp *Preprocessor) handleObjDefine(ident *Token, in chan *Token) {
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
