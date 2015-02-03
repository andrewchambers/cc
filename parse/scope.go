package parse

type scope struct {
	parent *scope
	kv     map[string]interface{}
}

func (s *scope) lookup(key string) (interface{}, error) {
	return nil, nil
}

func (s *scope) define(k string, v interface{}) error {
	return nil
}

func newScope(parent *scope) *scope {
	ret := &scope{}
	ret.parent = parent
	ret.kv = make(map[string]interface{})
	return ret
}
