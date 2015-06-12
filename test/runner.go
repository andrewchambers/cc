package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

var (
	filter = flag.String("filter", ".*", "A regex filtering which tests to run")
	cfg    = Config{
		CompileCmd:  "x64cc -o {{.Out}} {{.In}}",
		AssembleCmd: "as {{.In}} -o {{.Out}}",
		LinkCmd:     "gcc {{.In}} -o {{.Out}}",
	}
)

type Config struct {
	CompileCmd  string
	AssembleCmd string
	LinkCmd     string
}

func (c Config) Compile(in, out string) error {
	return RunWithInOutTemplate(in, out, c.CompileCmd, 5*time.Second)
}

func (c Config) Assemble(in, out string) error {
	return RunWithInOutTemplate(in, out, c.AssembleCmd, 5*time.Second)
}

func (c Config) Link(in, out string) error {
	return RunWithInOutTemplate(in, out, c.LinkCmd, 5*time.Second)
}

func RunWithInOutTemplate(in, out, templ string, timeout time.Duration) error {
	data := struct{ In, Out string }{In: in, Out: out}
	t := template.New("gencmdline")
	t, err := t.Parse(templ)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	err = t.Execute(&b, data)
	if err != nil {
		return err
	}
	cmdline := b.String()
	return RunWithTimeout(cmdline, timeout)
}

// True on success, else fail.
func RunWithTimeout(command string, timeout time.Duration) error {
	args := strings.Split(command, " ")
	if len(args) == 0 {
		return fmt.Errorf("malformed command %s", command)
	}
	bin := args[0]
	args = args[1:]
	c := exec.Command(bin, args...)
	rc := make(chan error)
	go func() {
		err := c.Run()
		rc <- err
	}()
	t := time.NewTicker(timeout)
	defer t.Stop()
	select {
	case <-t.C:
		return fmt.Errorf("%s timed out", bin)
	case err := <-rc:
		return err
	}
}

// Tests which are expected to run and return an error code true or false.
func ExecuteTests(tdir string) error {
	fmt.Println("execute tests in", tdir)
	passcount := 0
	runcount := 0
	tests, err := ioutil.ReadDir(tdir)
	if err != nil {
		panic(err)
	}
	for _, t := range tests {
		if !strings.HasSuffix(t.Name(), ".c") {
			continue
		}
		tc := filepath.Join(tdir, t.Name())
		m, err := regexp.MatchString(*filter, tc)
		if err != nil {
			panic(err)
		}
		if !m {
			continue
		}
		sname := tc + ".s"
		oname := tc + ".o"
		bname := tc + ".bin"
		runcount += 1
		err = cfg.Compile(tc, sname)
		if err != nil {
			fmt.Printf("FAIL: %s compile - %s\n", tc, err)
			continue
		}
		err = cfg.Assemble(sname, oname)
		if err != nil {
			fmt.Printf("FAIL: %s assemble - %s\n", tc, err)
			continue
		}
		err = cfg.Link(oname, bname)
		if err != nil {
			fmt.Printf("FAIL: %s link - %s\n", tc, err)
			continue
		}
		err = RunWithTimeout(bname, 5*time.Second)
		if err != nil {
			fmt.Printf("FAIL: %s execute - %s\n", tc, err)
			continue
		}
		fmt.Printf("PASS: %s\n", tc)
		passcount += 1
	}
	if passcount != runcount {
		return fmt.Errorf("passed %d/%d", passcount, runcount)
	}
	return nil
}

func main() {
	flag.Parse()
	pass := true
	for _, tdir := range []string{"test/testcases/execute", "test/testcases/bugs"} {
		err := ExecuteTests(tdir)
		if err != nil {
			fmt.Printf("%s FAIL: %s\n", tdir, err)
			pass = false
		}
	}
	if !pass {
		os.Exit(1)
	}
}
