package cpp

//This file defines hidesets. Each token has a hideset.
//The hideset of a token is the set of identifiers whose expansion resulted inthe token.
//Hidesets prevent infinite expansion by not rexpanding if its hideset contains the macro.
//Needs a copy method

type hideSet struct {
	kv map[string]struct{}
}

func (hs *hideSet) copy() *hideSet {
	if hs == nil {
		return nil
	}
	ret := newHideSet()
	for k := range hs.kv {
		ret.kv[k] = struct{}{}
	}
	return ret
}

func (hs *hideSet) put(tok *Token) {
	hs.kv[tok.Val] = struct{}{}
}

func (hs *hideSet) putTokList(tl *tokenList) {
	for e := tl.front(); e != nil; e = e.Next() {
		t := e.Value.(*Token)
		hs.kv[t.Val] = struct{}{}
	}
}

func (hs *hideSet) contains(val string) bool {
	if hs == nil {
		return false
	}
	_, ok := hs.kv[val]
	return ok
}

func newHideSet() *hideSet {
	ret := &hideSet{}
	ret.kv = make(map[string]struct{})
	return ret
}

func hideSetIntersection(a *hideSet, b *hideSet) *hideSet {
	ret := newHideSet()
	for k := range a.kv {
		if b.contains(k) {
			ret.kv[k] = struct{}{}
		}
	}
	return ret
}

func hideSetUnion(a *hideSet, b *hideSet) *hideSet {
	ret := newHideSet()
	for k := range a.kv {
		ret.kv[k] = struct{}{}
	}
	for k := range b.kv {
		ret.kv[k] = struct{}{}
	}
	return ret
}
