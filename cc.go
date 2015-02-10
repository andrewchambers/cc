package main

import (
	"flag"
	"fmt"
	"github.com/andrewchambers/cc/cpp"
	"github.com/andrewchambers/cc/parse"
	"io"
	"os"
	"runtime/pprof"
)

func printVersion() {
	fmt.Println("cc version 0.01")
}

func printUsage() {
	printVersion()
	fmt.Println()
	fmt.Println("This software is C compiler.")
	fmt.Println()
	fmt.Println("Software by Andrew Chambers 2014-2015 - andrewchamberss@gmail.com")
	fmt.Println()
	flag.PrintDefaults()
}

func compileFile(path string, includeDirs []string, out io.Writer) error {
	return nil
}

func preprocessFile(sourceFile string, out io.WriteCloser) {
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	lexer := cpp.Lex(sourceFile, f)
	pp := cpp.New(lexer, nil)
	for {
		tok, err := pp.Next()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(out, "%s:%s:%d:%d\n", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
		if tok.Kind == cpp.EOF {
			return
		}
	}
}

func parseFile(sourceFile string, out io.WriteCloser) {
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for parsing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	lexer := cpp.Lex(sourceFile, f)
	pp := cpp.New(lexer, nil)
	err = parse.Parse(pp)
	if err != nil {
	    fmt.Fprintln(os.Stderr, err)
	    os.Exit(1)
	}
}

func tokenizeFile(sourceFile string, out io.WriteCloser) {
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	lexer := cpp.Lex(sourceFile, f)
	for {
		tok, err := lexer.Next()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(out, "%s:%s:%d:%d\n", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
		if tok.Kind == cpp.EOF {
			return
		}
	}
}

func main() {
	flag.Usage = printUsage
	preprocessOnly := flag.Bool("E", false, "Preprocess only.")
	tokenizeOnly := flag.Bool("T", false, "Tokenize only (For debugging).")
	astOnly := flag.Bool("A", false, "Print AST (For compiler debugging).")
	doProfiling := flag.Bool("P", false, "Profile the compiler (For compiler debugging).")
	version := flag.Bool("version", false, "Print version info and exit.")
	outputPath := flag.String("o", "-", "File to write output to, - for stdout.")
	flag.Parse()

	if *doProfiling {
		profile, err := os.Create("ccrun.prof")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open profile file: %s\n", err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(profile)
		defer pprof.StopCPUProfile()
	}

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
		preprocessFile(input, output)
	} else if *tokenizeOnly {
		tokenizeFile(input, output)
	} else if *astOnly {
		parseFile(input, output)
	} else {
		compileFile(input, nil, output)
	}
}
