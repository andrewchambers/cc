package cpp

import "fmt"

//Data structures representing macros inside the cpreprocessor.
//These should be immutable.

type objMacro struct {
	tokens *tokenList
}

func newObjMacro(tokens *tokenList) *objMacro {
	return &objMacro{tokens}
}

type funcMacro struct {
	//Map of macro string to arg position
	nargs int
	//0 based index
	args map[string]int
	//Tokens of the macro.
	tokens *tokenList
}

//Returns if the token is an argument to the macro
//Also returns the zero based index of which argument it is.
//This corresponds to the position in the invocation list.
func (fm *funcMacro) isArg(t *Token) (int, bool) {
	v, ok := fm.args[t.Val]
	return v, ok
}

//args should be a list of ident tokens
func newFuncMacro(args *tokenList, tokens *tokenList) (*funcMacro, error) {
	ret := new(funcMacro)
	ret.nargs = 0
	ret.args = make(map[string]int)
	idx := 0
	for e := args.front(); e != nil; e = e.Next() {
		tok := e.Value.(*Token)
		_, ok := ret.args[tok.Val]
		if ok {
			return nil, fmt.Errorf("error duplicate argument " + tok.Val)
		}
		ret.args[tok.Val] = idx
		ret.nargs += 1
		idx += 1
	}
	ret.tokens = tokens
	return ret, nil
}
