package parse

import "cc/cpp"

func Parse(ts chan *cpp.Token) {
	adapter := newAdapter(ts)
	yyParse(adapter)
}
