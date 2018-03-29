package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"time"

	"github.com/semrush/zenrpc/parser"
)

const (
	version = "1.1.0"

	openIssueURL = "https://github.com/semrush/zenrpc/issues/new"
	githubURL    = "https://github.com/semrush/zenrpc"
)

var (
	useStringCase = flag.Bool("snake-case", false, "Generate names as snake_case instead of lower case.")
	verbose       = flag.Bool("v", true, "When false, allows you to omit service listing.")
	filename      = flag.String("file", "", "Is used for passing file name clearly and directly.")
)

func init() {
	flag.Parse()
	flag.CommandLine.SetOutput(os.Stdout)
	if *filename == "" {
		*filename = os.Getenv("GOFILE")
	}
}

func main() {
	start := time.Now()
	fmt.Printf("Generator version: %s\n", version)

	if *filename == "" {
		fmt.Fprintln(os.Stderr, "File path is empty")
		os.Exit(1)
	}

	fmt.Printf("Entrypoint: %s\n", *filename)

	pi := parser.NewPackageInfo(*useStringCase)
	if err := pi.Parse(*filename); err != nil {
		printError(err)
		os.Exit(1)
	}

	if len(pi.Services) == 0 {
		fmt.Fprintln(os.Stderr, "Services not found")
		os.Exit(1)
	}

	outputFileName, err := generateFile(pi)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s\n", outputFileName)
	fmt.Printf("Duration: %dms\n", int64(time.Since(start)/time.Millisecond))
	if *verbose {
		fmt.Println()
		fmt.Print(pi)
		fmt.Println()
	}
}

func printError(err error) {
	// print error to stderr
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)

	// print contact information to stdout
	fmt.Println("\nYou may help us and create issue:")
	fmt.Printf("\t%s\n", openIssueURL)
	fmt.Println("For more information, see:")
	fmt.Printf("\t%s\n\n", githubURL)
	fmt.Printf("Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func generateFile(pi *parser.PackageInfo) (string, error) {
	outputFileName := filepath.Join(pi.Dir, pi.PackageName+parser.GenerateFileSuffix)

	output := new(bytes.Buffer)
	if err := serviceTemplate.Execute(output, pi); err != nil {
		return outputFileName, err
	}

	source, err := format.Source(output.Bytes())
	if err != nil {
		return outputFileName, err
	}

	file, err := os.Create(outputFileName)
	if err != nil {
		return outputFileName, err
	}
	defer file.Close()

	if _, err = file.Write(source); err != nil {
		return outputFileName, err
	}

	return outputFileName, nil
}
