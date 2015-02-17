package parse

type scope struct {
	parent *scope
	kv     map[string]Symbol
}

func (s *scope) lookup(key string) (Symbol, error) {
	return nil, nil
}

func (s *scope) define(k string, v Symbol) error {
	return nil
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
	Init  Node
}
