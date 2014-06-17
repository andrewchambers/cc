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
	tokChan, errChan := Lex(cfile, f)

	for {
		expectedTokS := ""
		if scanner.Scan() {
			expectedTokS = scanner.Text()
		}
		select {
		case tok := <-tokChan:
			if tok == nil {
				if expectedTokS != "" {
					t.Errorf("Unexpected end of token stream. Expected %s", expectedTokS)
				}
				return
			}

			tokS := fmt.Sprintf("%s:%d:%d", tok.Val, tok.Pos.Line, tok.Pos.Col)
			if tokS != expectedTokS && !errorReported {
				if expectedTokS == "" {
					t.Errorf("Error while lexing %s - extra token %s", cfile, tokS)
				} else {
					t.Errorf("Error while lexing %s: %s != %s", cfile, tokS, expectedTokS)
				}
				errorReported = true
			}
		case err = <-errChan:
			t.Errorf("Testfile %s failed because %s", cfile, err)
			return
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
