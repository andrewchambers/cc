package cpp

import "container/list"

type tokenList struct {
	l *list.List
}

func newTokenList() *tokenList {
	return &tokenList{list.New()}
}

func (tl *tokenList) isEmpty() bool {
	return tl.l.Len() == 0
}

func (tl *tokenList) popFront() *Token {
	if tl.isEmpty() {
		panic("internal error")
	}
	fronte := tl.l.Front()
	ret := fronte.Value.(*Token)
	return ret
}

func (tl *tokenList) append(tok *Token) {
	tl.l.PushBack(tok)
}

func (tl *tokenList) appendList(toAdd *tokenList) {
	l := toAdd.l
	for e := l.Front(); e != nil; e = e.Next() {
		tl.l.PushBack(e.Value)
	}
}

func (tl *tokenList) front() *list.Element {
	if tl.isEmpty() {
		panic("internal error")
	}
	return tl.l.Front()
}
