package cpp

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
	//0 based indexe
	args map[string]int
	//Tokens of the macro.
	tokens *tokenList
}

//args should be a list of ident tokens
func newFuncMacro(args *tokenList, tokens *tokenList) *funcMacro {
	ret := new(funcMacro)
	ret.args = make(map[string]int)
	idx := 0
	for e := args.front(); e != nil; e = e.Next() {
		tok := e.Value.(*Token)
		ret.args[tok.Val] = idx
		idx += 1
	}
	ret.tokens = tokens
	return ret
}
