package main

import (
	"fmt"
	"os"
	"flag"
)

func main() {
	//Control the number of parameters sent
	if len(os.Args) <= 1 {
		fmt.Println("Usage: etcd-walker [-ls | -cp | -grep | ...] ")
		os.Exit(1)
	}
	// Define the flags and their associated variables
	var (
		lsFlag      string
		cpSource    string
		grepPattern string
		outputPath  string
	)

	flag.StringVar(&lsFlag, "ls", "", "List etcds")
	flag.StringVar(&cpSource, "cp", "", "Copy source etcd")
	flag.StringVar(&grepPattern, "grep", "", "Search for pattern")
	flag.StringVar(&outputPath, "out", "default.log", "Output etcd path")

	// Parse the command-line arguments
	flag.Parse()

	// Basic validation: Ensure at least one primary action is requested
	if lsFlag == "" && cpSource == "" && grepPattern == "" {
		fmt.Println("Error: Must provide one of ls, cp <source>, or grep <pattern>")
		flag.Usage()
		os.Exit(1)
	}

	// More specific validation for arguments requiring values
	if flag.Lookup("cp").Value.String() != "" && cpSource == "" {
		fmt.Println("Error: The --cp command requires a source etcd value.")
		flag.Usage()
		os.Exit(1)
	}

	if flag.Lookup("grep").Value.String() != "" && grepPattern == "" {
		fmt.Println("Error: The --grep command requires a search pattern value.")
		flag.Usage()
		os.Exit(1)
	}

	// Process the arguments and their values
	//Implement the functions ls,cp,grep
	if lsFlag != "" {
		fmt.Println("Listing etcd prefix...")
		lsFlagVal := flag.Lookup("ls").Value.String()
		fmt.Println("Value passed to ls",lsFlagVal)
		// Do the rest here
		// Do I want to implement a logic here on the value ?
	}

	if cpSource != "" {
		fmt.Printf("Copying etcd: %s to ... \n", cpSource)
		// 	// Do the rest here for --cp with cpSource value
	}

	if grepPattern != "" {
		fmt.Printf("Searching for pattern: '%s' in ... \n", grepPattern)
		// 	// Do the rest here --grep with grepPattern value
	}

	fmt.Printf("Output will be written to: %s\n", outputPath)
	
}