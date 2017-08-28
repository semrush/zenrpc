package main

import (
	"bytes"
	"fmt"
	"github.com/semrush/zenrpc/parser"
	"go/format"
	"os"
	"path/filepath"
	"time"
)

const (
	openIssueURL = "https://github.com/semrush/zenrpc/issues/new"
	githubURL    = "https://github.com/semrush/zenrpc"
)

func main() {
	start := time.Now()

	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[len(os.Args)-1]
	} else {
		filename = os.Getenv("GOFILE")
	}

	if len(filename) == 0 {
		fmt.Fprintln(os.Stderr, "File path is empty")
		os.Exit(1)
	}

	fmt.Printf("Entrypoint: %s\n", filename)

	pi := parser.NewPackageInfo()
	dir, err := pi.ParseFiles(filename)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if len(pi.Services) == 0 {
		fmt.Fprintln(os.Stderr, "Services not found")
		os.Exit(1)
	}

	outputFileName, err := generateFile(pi, dir)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s\n", outputFileName)
	fmt.Printf("Duration: %dms\n", int64(time.Since(start)/time.Millisecond))
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
