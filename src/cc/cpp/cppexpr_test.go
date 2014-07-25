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
	{"(2)", 2, false},
	{"(-2)", -2, false},
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
	{"0 || 0", 0, false},
	{"1 || 0", 1, false},
	{"0 || 1", 1, false},
	{"1 || 1", 1, false},
	{"0 && 0", 0, false},
	{"1 && 0", 0, false},
	{"0 && 1", 0, false},
	{"1 && 1", 1, false},
	{"0xf0 | 1", 0xf1, false},
	{"0xf0 & 1", 0, false},
	{"0xf0 & 0x1f", 0x10, false},
	{"1 ^ 1", 0, false},
	{"1 == 1", 1, false},
	{"1 == 0", 0, false},
	{"1 != 1", 0, false},
	{"0 != 1", 1, false},
	{"0 > 1", 0, false},
	{"0 < 1", 1, false},
	{"0 > -1", 1, false},
	{"0 < -1", 0, false},
	{"0 >= 1", 0, false},
	{"0 <= 1", 1, false},
	{"0 >= -1", 1, false},
	{"0 <= -1", 0, false},
	{"0 < 0", 0, false},
	{"0 <= 0", 1, false},
	{"0 > 0", 0, false},
	{"0 >= 0", 1, false},
	{"1 << 1", 2, false},
	{"2 >> 1", 1, false},
	{"2 + 1", 3, false},
	{"2 - 3", -1, false},
	{"2 * 3", 6, false},
	{"6 / 3", 2, false},
	{"7 % 3", 1, false},
	{"0,1", 1, false},
	{"1,0", 0, false},
	{"2+2*3+2", 10, false},
	{"(2+2)*(3+2)", 20, false},
	{"2 + 2 + 2 + 2 == 2 + 2 * 3", 1, false},
	{"0 ? 1 : 2", 2, false},
	{"1 ? 1 : 2", 1, false},
	{"(1 ? 1 ? 1337 : 1234 : 2) == 1337", 1, false},
	{"(1 ? 0 ? 1337 : 1234 : 2) == 1234", 1, false},
	{"(0 ? 1 ? 1337 : 1234 : 2) == 2", 1, false},
	{"(0 ? 1 ? 1337 : 1234 : 2 ? 3 : 4) == 3", 1, false},
	{"0 , 1 ? 1 , 0 : 2  ", 0, false},
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
			t.Errorf("test %s failed - got %d expected %d", tc.expr, result, tc.expected)
		}
	}
}

func BenchmarkCPPExprParsing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		expr := "1 + 2 * 3 /2 + 4 ^ 3 - -2 * (3 % 2) * 2 + 3 + 0x1234 - 0x1234"
		r := bytes.NewBufferString(expr)
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
		//b.ResetTimer()
		result := evalIfExpr(isDefined, nextTok, onErr)
		if e != nil || result != 2 {
			b.FailNow()
		}
	}
}
