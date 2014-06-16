package main

import (
	"cc/cpp"
	"flag"
	"fmt"
	"io"
	"os"
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

func preprocessFile(sourceFile string, out io.Writer) {
	_, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	_ = cpp.New(nil)
}

func tokenizeFile(sourceFile string, out io.Writer) {
	f, err := os.Open(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open source file %s for preprocessing: %s\n", sourceFile, err)
		os.Exit(1)
	}
	//Don't care about cancelling the lexing here.
	errChan, tokChan := cpp.Lex(sourceFile, f, make(chan struct{}))
	for t := range tokChan {
		fmt.Println(*t)
	}
	err = <-errChan
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func main() {
	flag.Usage = printUsage
	preprocessOnly := flag.Bool("E", false, "Preprocess only")
	tokenizeOnly := flag.Bool("T", false, "Tokenize only (For debugging).")
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
	var output io.Writer
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
	} else {
		CompileFile(input, nil, output)
	}
}
