package parse

import (
	"github.com/andrewchambers/cc/cpp"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func parseTestCase(t *testing.T, path string) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	lexer := cpp.Lex(path, f)
	pp := cpp.New(lexer, nil)
	err = Parse(pp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParser(t *testing.T) {
	info, err := ioutil.ReadDir("parsetests")
	if err != nil {
		t.Fatal(err)
	}
	for i := range info {
		filename := info[i].Name()
		if !strings.HasSuffix(filename, ".c") {
			continue
		}
		parseTestCase(t, "parsetests/"+filename)
	}
}
