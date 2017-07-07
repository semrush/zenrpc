package main

import (
	"bytes"
	"fmt"
	"github.com/sergeyfast/zenrpc/parser"
	"go/format"
	"os"
	"path/filepath"
	"time"
)

const (
	openIssueURL = "https://github.com/sergeyfast/zenrpc/issues/new"
	githubURL    = "https://github.com/sergeyfast/zenrpc"
)

func main() {
	start := time.Now()

	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[len(os.Args)-1]
	} else {
		filename = os.Getenv("GOFILE")
	}

	fmt.Printf("Entrypoint: %s\n", filename)

	pi := parser.PackageInfo{Services: []*parser.Service{}, Errors: make(map[int]string)}
	dir, err := pi.ParseFiles(filename)
	if err != nil {
		printError(err)
	}

	if len(pi.Services) == 0 {
		fmt.Printf("Services not found")
		return
	}

	outputFileName, err := generateFile(&pi, dir)
	if err != nil {
		printError(err)
	}

	fmt.Printf("Generated: %s\n", outputFileName)
	fmt.Printf("Duration: %s\n", time.Since(start))
	fmt.Println()
	fmt.Print(pi)
	fmt.Println()
}

func printError(err error) {
	// print error to stderr
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)

	// print contact information to stdout
	fmt.Println("\nYou may help us and create issue:")
	fmt.Printf("\t%s\n", openIssueURL)
	fmt.Println("For more information, see:")
	fmt.Printf("\t%s\n\n", githubURL)

	os.Exit(1)
}

func generateFile(pi *parser.PackageInfo, dir string) (string, error) {
	outputFileName := filepath.Join(dir, pi.PackageName+parser.GenerateFileSuffix)
	file, err := os.Create(outputFileName)
	if err != nil {
		return outputFileName, err
	}
	defer file.Close()

	output := new(bytes.Buffer)
	if err := serviceTemplate.Execute(output, pi); err != nil {
		return outputFileName, err
	}

	source, err := format.Source(output.Bytes())
	if err != nil {
		return outputFileName, err
	}

	if _, err = file.Write(source); err != nil {
		return outputFileName, err
	}

	return outputFileName, nil
}
