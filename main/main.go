package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/notJoon/rmlog"
)

func main() {
	filePath := flag.String("file", "", "Path to the file to process")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Please provide the path to the file using -file flag.")
		flag.Usage()
		os.Exit(1)
	}

	err := rmlog.ProcessFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("File processed successfully.")
}
