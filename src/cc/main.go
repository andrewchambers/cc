package main

import (
	"cc/cpp"
	"cc/parse"
	"flag"
	"fmt"
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
	fmt.Println("This software is a portable C compiler.")
	fmt.Println("It was created with the goals of being the small and hackable.")
	fmt.Println("It is hopefully one of the easiest C compilers to port and understand.")
	fmt.Println()
	fmt.Println("Software by Andrew Chambers 2014 - andrewchamberss@gmail.com")
	fmt.Println()
	flag.PrintDefaults()
}

func compileFile(path string, includeDirs []string, out io.Writer) error {
	return nil
}

func preprocessFile(sourceFile string, out io.WriteCloser) {
	defer out.Close()
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	lexTokChan := cpp.Lex(sourceFile, f)
	pp := cpp.New(nil)
	ppTokChan := pp.Preprocess(lexTokChan)
	for tok := range ppTokChan {
		if tok == nil {
			return
		}
		if tok.Kind == cpp.ERROR {
			fmt.Fprintln(os.Stderr, tok.Val)
			os.Exit(1)
		}
		fmt.Fprintf(out, "%s:%s:%d:%d\n", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
	}
}

func parseFile(sourceFile string, out io.WriteCloser) {
	defer out.Close()
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for parsing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	lexTokChan := cpp.Lex(sourceFile, f)
	pp := cpp.New(nil)
	ppTokChan := pp.Preprocess(lexTokChan)
	parse.Parse(ppTokChan)
}

func tokenizeFile(sourceFile string, out io.WriteCloser) {
	defer out.Close()
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	tokChan := cpp.Lex(sourceFile, f)
	for tok := range tokChan {
		if tok == nil {
			return
		}
		if tok.Kind == cpp.ERROR {
			fmt.Fprintln(os.Stderr, tok.Val)
			os.Exit(1)
		}
		fmt.Fprintf(out, "%s:%s:%d:%d\n", tok.Kind, tok.Val, tok.Pos.Line, tok.Pos.Col)
	}
}

func main() {
	flag.Usage = printUsage
	preprocessOnly := flag.Bool("E", false, "Preprocess only")
	tokenizeOnly := flag.Bool("T", false, "Tokenize only (For debugging).")
	parseOnly := flag.Bool("A", false, "Print AST (For debugging).")
	doProfiling := flag.Bool("P", false, "Profile the compiler (For debugging).")
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
	} else if *parseOnly {
		parseFile(input, output)
	} else {
		compileFile(input, nil, output)
	}
}
