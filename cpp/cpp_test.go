package cpp

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func sourceToExpectFile(s string) string {
	return s[0:len(s)-2] + ".exp"
}

func lexTestCase(t *testing.T, cfile string, expectfile string) {
	f, err := os.Open(cfile)
	if err != nil {
		t.Fatal(err)
	}
	ef, err := os.Open(expectfile)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(ef)
	errorReported := false
	lexer := Lex(cfile, f)
	for {
		expectedTokS := ""
		if scanner.Scan() {
			expectedTokS = scanner.Text()
		}
		tok, err := lexer.Next()
		if err != nil {
			t.Errorf("Testfile %s failed because %s", cfile, err)
			return
		}
		tokS := fmt.Sprintf("%s:%s:%d:%d", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
		if tokS != expectedTokS && !errorReported {
			if expectedTokS == "" {
				t.Errorf("Test failed %s - extra token %s", cfile, tokS)
			} else {
				t.Errorf("Test failed %s: got %s expected %s ", cfile, tokS, expectedTokS)
			}
			errorReported = true
		}
		if tok.Kind == EOF {
			break
		}
	}
}

func TestLexer(t *testing.T) {
	info, err := ioutil.ReadDir("lextests")
	if err != nil {
		t.Fatal(err)
	}
	for i := range info {
		filename := info[i].Name()
		if !strings.HasSuffix(filename, ".c") {
			continue
		}
		expectPath := sourceToExpectFile(filename)
		lexTestCase(t, "lextests/"+filename, "lextests/"+expectPath)
	}
}

func cppTestCase(t *testing.T, cfile string, expectfile string) {
	f, err := os.Open(cfile)
	if err != nil {
		t.Fatal(err)
	}
	ef, err := os.Open(expectfile)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(ef)
	errorReported := false
	pp := New(Lex(cfile, f), NewStandardIncludeSearcher("./cpptests/"))
	for {
		expectedTokS := ""
		if scanner.Scan() {
			expectedTokS = scanner.Text()
		}
		tok, err := pp.Next()
		if err != nil {
			t.Errorf("Testfile %s failed because %s", cfile, err)
			return
		}
		tokS := fmt.Sprintf("%s:%s:%d:%d", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
		if tokS != expectedTokS && !errorReported {
			if expectedTokS == "" {
				t.Errorf("Test failed %s - extra token %s", cfile, tokS)
			} else {
				t.Errorf("Test failed %s: got %s expected %s ", cfile, tokS, expectedTokS)
			}
			errorReported = true
		}
		if tok.Kind == EOF {
			break
		}
	}
}

func TestPreprocessor(t *testing.T) {
	info, err := ioutil.ReadDir("cpptests")
	if err != nil {
		t.Fatal(err)
	}
	for i := range info {
		filename := info[i].Name()
		if !strings.HasSuffix(filename, ".c") {
			continue
		}
		expectPath := sourceToExpectFile(filename)
		cppTestCase(t, "cpptests/"+filename, "cpptests/"+expectPath)
	}
}
