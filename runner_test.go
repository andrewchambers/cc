package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestCC(t *testing.T) {
	files, err := ioutil.ReadDir("test")
	if err != nil {
		t.Fatal(err)
	}
	for _, finf := range files {
		if finf.IsDir() {
			continue
		}
		if !strings.HasSuffix(finf.Name(), ".c") {
			continue
		}
		sfile, err := os.Create("test/" + finf.Name() + ".s")
		if err != nil {
			t.Fatal(err)
		}
		err = compileFile("test/"+finf.Name(), sfile)
		if err != nil {
			t.Fatal(err)
		}
	}
}
