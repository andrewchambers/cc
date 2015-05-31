package main

import (
	"flag"
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"github.com/andrewchambers/cc/parse"
	"github.com/andrewchambers/cc/report"
	"io"
	"os"
)

func printVersion() {
	fmt.Println("x64cc")
}

func printUsage() {
	printVersion()
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  x64cc [FLAGS] FILE.c")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  CCDEBUG=true enables extended error messages for debugging the compiler.")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Software by Andrew Chambers 2014-2015 - andrewchamberss@gmail.com")
}

func compileFile(path string, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("Failed to open source file %s for parsing: %s\n", path, err)
		return err
	}
	lexer := cpp.Lex(path, f)
	pp := cpp.New(lexer, nil)
	toplevels, err := parse.Parse(x64SzDesc, pp)
	if err != nil {
		return err
	}
	return Emit(toplevels, out)
}

func main() {
	flag.Usage = printUsage
	version := flag.Bool("version", false, "Print version info and exit.")
	outputPath := flag.String("o", "-", "Write output to `file`, '-' for stdout.")
	flag.Parse()
	if *version {
		printVersion()
		return
	}
	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Bad number of args, please specify a single source file.\n")
		os.Exit(1)
	}
	input := flag.Args()[0]
	var output io.WriteCloser
	var err error
	if *outputPath == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open output file %s\n", err)
			os.Exit(1)
		}
	}
	err = compileFile(input, output)
	if err != nil {
		report.ReportError(err)
	}
}
