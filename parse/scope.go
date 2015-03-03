package parse

import "fmt"

type scope struct {
	parent *scope
	kv     map[string]Symbol
}

func (s *scope) lookup(k string) (Symbol, error) {
	sym, ok := s.kv[k]
	if ok {
		return sym, nil
	}
	if s.parent != nil {
		return s.parent.lookup(k)
	}
	return nil, fmt.Errorf("%s is not defined", k)
}

func (s *scope) define(k string, v Symbol) error {
	_, ok := s.kv[k]
	if ok {
		return fmt.Errorf("redefinition of %s", k)
	}
	s.kv[k] = v
	return nil
}

func (s *scope) String() string {
	str := ""
	if s.parent != nil {
		str += s.parent.String() + "\n"
	}
	str += fmt.Sprintf("%v", s.kv)
	return str
}

func newScope(parent *scope) *scope {
	ret := &scope{}
	ret.parent = parent
	ret.kv = make(map[string]Symbol)
	return ret
}

type Symbol interface{}

type GSymbol struct {
	Label string
	Type  CType
}

type LSymbol struct {
	Type CType
}

type TSymbol struct {
	Type CType
}
