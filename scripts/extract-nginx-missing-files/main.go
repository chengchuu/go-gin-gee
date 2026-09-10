package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const maxLogLineSize = 4 * 1024 * 1024

var missingFilePattern = regexp.MustCompile(`open\(\) "([^"\r\n]+)" failed \(2: No such file or directory\)(?:,|$)`)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("extract-nginx-missing-files", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var inputPath string
	var outputPath string
	flags.StringVar(&inputPath, "in", "", "input Nginx error log (default: stdin)")
	flags.StringVar(&outputPath, "out", "", "output path list (default: stdout)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [-in path] [-out path]\n", flags.Name())
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", flags.Args())
		flags.Usage()
		return 2
	}

	if inputPath != "" && outputPath != "" {
		same, err := sameFile(inputPath, outputPath)
		if err != nil {
			fmt.Fprintf(stderr, "failed to compare input and output paths: %v\n", err)
			return 1
		}
		if same {
			fmt.Fprintln(stderr, "input and output must be different files")
			return 2
		}
	}

	input := stdin
	var inputFile *os.File
	if inputPath != "" {
		var err error
		inputFile, err = os.Open(inputPath)
		if err != nil {
			fmt.Fprintf(stderr, "failed to open input file: %v\n", err)
			return 1
		}
		defer inputFile.Close()
		input = inputFile
	}

	output := stdout
	var outputFile *os.File
	if outputPath != "" {
		var err error
		outputFile, err = os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(stderr, "failed to create output file: %v\n", err)
			return 1
		}
		output = outputFile
	}

	processErr := extractMissingPaths(input, output)
	if outputFile != nil {
		if err := outputFile.Close(); processErr == nil {
			processErr = err
		}
	}
	if processErr != nil {
		fmt.Fprintf(stderr, "failed to process Nginx error log: %v\n", processErr)
		return 1
	}
	return 0
}

func extractMissingPaths(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), maxLogLineSize)
	writer := bufio.NewWriter(output)
	seen := make(map[string]struct{})

	for scanner.Scan() {
		matches := missingFilePattern.FindStringSubmatch(scanner.Text())
		if matches == nil {
			continue
		}

		path := matches[1]
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		if _, err := fmt.Fprintln(writer, path); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return writer.Flush()
}

func sameFile(inputPath, outputPath string) (bool, error) {
	absoluteInput, err := filepath.Abs(inputPath)
	if err != nil {
		return false, err
	}
	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return false, err
	}
	if filepath.Clean(absoluteInput) == filepath.Clean(absoluteOutput) {
		return true, nil
	}

	inputInfo, inputErr := os.Stat(inputPath)
	outputInfo, outputErr := os.Stat(outputPath)
	if inputErr == nil && outputErr == nil {
		return os.SameFile(inputInfo, outputInfo), nil
	}
	if inputErr != nil && !os.IsNotExist(inputErr) {
		return false, inputErr
	}
	if outputErr != nil && !os.IsNotExist(outputErr) {
		return false, outputErr
	}
	return false, nil
}
