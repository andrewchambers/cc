package cpp

import (
	"bytes"
	"testing"
)

var exprTestCases = []struct {
	expr      string
	expected  int64
	expectErr bool
}{
	{"1", 1, false},
	{"2", 2, false},
	{"0x1", 0x1, false},
	{"0x1", 0x1, false},
	{"-1", -1, false},
	{"-2", -2, false},
	{"0x1234", 0x1234, false},
	{"foo", 1, false},
	{"bang", 0, false},
	{"defined foo", 1, false},
	{"defined bang", 0, false},
	{"defined(foo)", 1, false},
	{"defined(bang)", 0, false},
	{"defined", 0, true},
	{"defined(bang", 0, true},
	{"defined bang)", 0, true},
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
			if !tc.expectErr {
				t.Errorf("test %s failed - got error <%s>", tc.expr, e)
			}
		} else if tc.expectErr {
			t.Errorf("test %s failed - expected an error", tc.expr)
		} else if result != tc.expected {
			t.Errorf("test %s failed - got %s expected %s", tc.expr, result, tc.expected)
		}
	}
}
