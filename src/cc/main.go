package main

import "fmt"
import "flag"

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

func main() {
	preprocessOnly := flag.Bool("E", false, "Preprocess only")
	version := flag.Bool("version", false, "Print version info and exit.")
	help := flag.Bool("h", false, "Print help message and exit.")
	outputFile := flag.String("o", "-", "File to write output to, - for stdout.")
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	if *version {
		printVersion()
	}

	fmt.Println("Arguments: ", *preprocessOnly, *outputFile)
}
