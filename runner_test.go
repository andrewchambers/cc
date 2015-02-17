package main

import (
	"io/ioutil"
	"os"
	"os/exec"
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
		tpath := "test/" + finf.Name()
		spath := "test/" + finf.Name() + ".s"
		bpath := "test/" + finf.Name() + ".bin"
		sfile, err := os.Create(spath)
		if err != nil {
			t.Fatal(err)
		}
		err = compileFile(tpath, sfile)
		if err != nil {
			t.Fatalf("compiling %s failed. %s", tpath, err)
		}
		gccout, err := exec.Command("gcc", spath, "-o", bpath).CombinedOutput()
		if err != nil {
			t.Log(string(gccout))
			t.Fatalf("assembling %s failed. %s", spath, err)
		}
		bout, err := exec.Command(bpath).CombinedOutput()
		if err != nil {
			t.Log(string(bout))
			t.Fatalf("running %s failed. %s", bpath, err)
		}
		t.Logf("%s OK", tpath)
	}
}
