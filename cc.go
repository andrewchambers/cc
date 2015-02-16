package main

import (
	"flag"
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"github.com/andrewchambers/cc/emit"
	"github.com/andrewchambers/cc/parse"
	"io"
	"os"
)

func printVersion() {
	fmt.Println("cc version 0.01")
}

func printUsage() {
	printVersion()
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cc [FLAGS] FILE.c")
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
	toplevels, err := parse.Parse(pp)
	if err != nil {
		return err
	}
	return emit.Emit(toplevels, out)
}

func preprocessFile(sourceFile string, out io.Writer) error {
	f, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
	}
	lexer := cpp.Lex(sourceFile, f)
	pp := cpp.New(lexer, nil)
	for {
		tok, err := pp.Next()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s:%s:%d:%d\n", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
		if tok.Kind == cpp.EOF {
			return nil
		}
	}
}

func tokenizeFile(sourceFile string, out io.Writer) error {
	f, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
	}
	lexer := cpp.Lex(sourceFile, f)
	for {
		tok, err := lexer.Next()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s:%s:%d:%d\n", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
		if tok.Kind == cpp.EOF {
			return nil
		}
	}
}

func main() {
	flag.Usage = printUsage
	preprocessOnly := flag.Bool("P", false, "Print tokens after preprocessing (For debugging).")
	tokenizeOnly := flag.Bool("T", false, "Print tokens after lexing (For debugging).")
	version := flag.Bool("version", false, "Print version info and exit.")
	outputPath := flag.String("o", "-", "File to write output to, - for stdout.")
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

	if *preprocessOnly {
		err := preprocessFile(input, output)
		reportError(err)
	} else if *tokenizeOnly {
		err := tokenizeFile(input, output)
		reportError(err)
	} else {
		err := compileFile(input, output)
		reportError(err)
	}
}
