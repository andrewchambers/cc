package parse

import "github.com/andrewchambers/cc/cpp"

type parser struct {
	types *scope
	decls *scope
}

func Parse(<-chan *cpp.Token) {
	parser := &parser{}
	parser.types = newScope(nil)
	parser.decls = newScope(nil)
}
