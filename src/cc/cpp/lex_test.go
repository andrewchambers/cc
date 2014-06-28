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

func performLexTestCase(t *testing.T, cfile string, expectfile string) {
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
	tokChan := Lex(cfile, f)
	for {
		expectedTokS := ""
		if scanner.Scan() {
			expectedTokS = scanner.Text()
		}
		tok := <-tokChan
		//t.Log(expectedTokS)
		if tok == nil {
			if expectedTokS != "" {
				t.Errorf("Unexpected end of token stream. Expected %s", expectedTokS)
			}
			return
		}
		if tok.Kind == ERROR {
			t.Errorf("Testfile %s failed because %s", cfile, tok.Val)
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
	}
}

func TestLexer(t *testing.T) {
	info, err := ioutil.ReadDir("lextestdata")
	if err != nil {
		t.Fatal(err)
	}
	for i := range info {
		filename := info[i].Name()
		if !strings.HasSuffix(filename, ".c") {
			continue
		}
		expectPath := sourceToExpectFile(filename)
		performLexTestCase(t, "lextestdata/"+filename, "lextestdata/"+expectPath)
	}
}
