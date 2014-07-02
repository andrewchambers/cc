package cpp

//This file defines hidesets. Each token has a hideset.
//The hideset of a token is the set of identifiers whose expansion resulted inthe token.
//Hidesets prevent infinite expansion by not rexpanding if its hideset contains the macro.

type hideSet struct {
	kv map[string]struct{}
}

func (hs *hideSet) put(val string) {
	hs.kv[val] = struct{}{}
}

func (hs *hideSet) contains(val string) bool {
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
			ret.put(k)
		}
	}
	return ret
}

func hideSetUnion(a *hideSet, b *hideSet) *hideSet {
	ret := newHideSet()
	for k := range a.kv {
		ret.put(k)
	}
	for k := range b.kv {
		ret.put(k)
	}
	return ret
}
