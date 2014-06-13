package main

import ( 
	"cc/cpp"
	"fmt"
	"flag"
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
	cpp.
}

func main() {
	flag.Usage = printUsage
	preprocessOnly := flag.Bool("E", false, "Preprocess only")
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
		fmt.Println("Bad number of args, please specify a single source file.")
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
			fmt.Errorf("Failed to open output file %s", err)
			os.Exit(1)
		}
	}

	if *preprocessOnly {
		preprocessOnly
	} else {
		CompileFile(input, nil, output)
	}
}
