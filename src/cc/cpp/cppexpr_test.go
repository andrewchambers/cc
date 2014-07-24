package cpp

import (
	"bytes"
	"testing"
)

var exprTestCases = []struct {
	expr     string
	expected int64
}{
	{"1", 1},
	{"2", 2},
	{"0x1", 0x1},
	{"0x1", 0x1},
	{"-1", -1},
	{"-2", -2},
	{"0x1234", 0x1234},
	{"foo", 1},
	{"bang", 0},
	{"defined foo", 1},
	{"defined bang", 0},
	{"defined(foo)", 1},
	{"defined(bang)", 0},
}

var testExprPredefined = map[string]struct{}{
	"foo": {},
	"bar": {},
	"baz": {},
}

func TestExprEval(t *testing.T) {
	for idx := range exprTestCases {
		tc := &exprTestCases[idx]
		r := bytes.NewBufferString(tc.expr)
		tokChan := Lex("testcase.c", r)

		var e error = nil
		onErr := func(evalErr error) {
			e = evalErr
		}
		nextTok := func() *Token {
			t := <-tokChan
			return t
		}
		isDefined := func(s string) bool {
			_, ok := testExprPredefined[s]
			return ok
		}

		result := evalIfExpr(isDefined, nextTok, onErr)
		if e != nil {
			t.Errorf("test %s failed - got error <%s>", tc.expr, e)
		} else if result != tc.expected {
			t.Errorf("test %s failed - got %s expected %s", tc.expr, result, tc.expected)
		}
	}
}
