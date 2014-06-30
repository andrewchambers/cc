package cpp

import (
	"fmt"
	"io"
)

type IncludeSearcher interface {
	//IncludeQuote is invoked when the preprocessor
	//encounters an include of the form #include "foo.h".
	IncludeQuote(path string) (io.Reader, error)
	//IncludeAngled is invoked when the preprocessor
	//encounters an include of the form #include <foo.h>.
	IncludeAngled(path string) (io.Reader, error)
}

type Preprocessor struct {
	is  IncludeSearcher
	out chan *Token
}

type StandardIncludeSearcher struct {
	//Priority order list of paths to search for headers
	systemHeaders []string
	localHeaders  []string
}

func (is *StandardIncludeSearcher) IncludeQuote(path string) (io.Reader, error) {
	return nil, fmt.Errorf("dummy include search.")
}

func (is *StandardIncludeSearcher) IncludeAngled(path string) (io.Reader, error) {
	return nil, fmt.Errorf("dummy include search.")
}

func New(is IncludeSearcher) *Preprocessor {
	ret := &Preprocessor{is: is}
	return ret
}

func (pp *Preprocessor) cppError(e string, pos FilePos) {
	emsg := fmt.Sprintf("Preprocessor error %s at %s", e, pos)
	pp.out <- &Token{Kind: ERROR, Val: emsg, Pos: pos}
	close(pp.out)
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
	for tok := range in {
		if tok.Kind == ERROR {
			pp.out <- tok
			panic(&breakout{})
		}

		switch tok.Kind {
		case DIRECTIVE:
			pp.handleDirective(tok, in)
		default:
			pp.out <- tok
		}

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
	//case "define":
	case "include":
		pp.handleInclude()
	//case "error":
	//case "warning":
	default:
		pp.cppError(fmt.Sprintf("unknown directive error %s", dirTok), dirTok.Pos)
	}
}

func (pp *Preprocessor) handleInclude() {
	tok := <-pp.out
	if tok.Kind != HEADER {
		pp.cppError("expected a header at %s", tok.Pos)
	}
	headerStr := tok.Val
	path := headerStr[1 : len(headerStr)-1]

	switch headerStr[0] {
	case '<':
		pp.is.IncludeAngled(path)
	case '"':
		pp.is.IncludeAngled(path)
	default:
		pp.cppError("internal error %s", tok.Pos)
	}
}

//Define can be used to predefine values in the preprocessor.
//This is what is used to perform -D defines from the command line.
func (pp *Preprocessor) Define() {

}
