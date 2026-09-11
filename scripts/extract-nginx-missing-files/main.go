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
	"sort"
	"time"
)

const (
	maxLogLineSize  = 4 * 1024 * 1024
	nginxTimeLayout = "2006/01/02 15:04:05"
)

var (
	missingFilePattern = regexp.MustCompile(`(?:^|[[:space:]])open\(\) "([^"\r\n]+)" failed \(2: No such file or directory\)(?:,|$)`)
	timestampPattern   = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})(?:[[:space:]]|$)`)
)

type extractOptions struct {
	sortNaturally bool
	since         *time.Time
	until         time.Time
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runAt(args, stdin, stdout, stderr, time.Now())
}

func runAt(args []string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int {
	flags := flag.NewFlagSet("extract-nginx-missing-files", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var inputPath string
	var outputPath string
	var sortNaturally bool
	var sinceValue string
	flags.StringVar(&inputPath, "in", "", "input Nginx error log (default: stdin)")
	flags.StringVar(&outputPath, "out", "", "output path list (default: stdout)")
	flags.BoolVar(&sortNaturally, "sort", false, "sort output paths naturally")
	flags.StringVar(&sinceValue, "since", "", "include entries since this local time (YYYY/MM/DD HH:MM:SS)")
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

	var since *time.Time
	if sinceValue != "" {
		parsed, err := time.ParseInLocation(nginxTimeLayout, sinceValue, now.Location())
		if err != nil {
			fmt.Fprintf(stderr, "invalid -since value %q: expected YYYY/MM/DD HH:MM:SS\n", sinceValue)
			return 2
		}
		if parsed.After(now) {
			fmt.Fprintln(stderr, "invalid -since value: start time must not be later than the current time")
			return 2
		}
		since = &parsed
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

	options := extractOptions{
		sortNaturally: sortNaturally,
		since:         since,
		until:         now,
	}
	processErr := extractMissingPaths(input, output, options)
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

func extractMissingPaths(input io.Reader, output io.Writer, options extractOptions) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), maxLogLineSize)
	seen := make(map[string]struct{})
	var paths []string

	for scanner.Scan() {
		line := scanner.Text()
		matches := missingFilePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		if options.since != nil {
			timestampMatch := timestampPattern.FindStringSubmatch(line)
			if timestampMatch == nil {
				continue
			}
			timestamp, err := time.ParseInLocation(nginxTimeLayout, timestampMatch[1], options.since.Location())
			if err != nil || timestamp.Before(*options.since) || timestamp.After(options.until) {
				continue
			}
		}

		path := matches[1]
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if options.sortNaturally {
		sort.Slice(paths, func(i, j int) bool {
			return naturalLess(paths[i], paths[j])
		})
	}

	writer := bufio.NewWriter(output)
	for _, path := range paths {
		if _, err := fmt.Fprintln(writer, path); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func naturalLess(left, right string) bool {
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(left) && rightIndex < len(right) {
		if isASCIIDigit(left[leftIndex]) && isASCIIDigit(right[rightIndex]) {
			leftEnd := digitRunEnd(left, leftIndex)
			rightEnd := digitRunEnd(right, rightIndex)
			leftNumber := trimLeadingZeros(left[leftIndex:leftEnd])
			rightNumber := trimLeadingZeros(right[rightIndex:rightEnd])

			if len(leftNumber) != len(rightNumber) {
				return len(leftNumber) < len(rightNumber)
			}
			if leftNumber != rightNumber {
				return leftNumber < rightNumber
			}
			leftIndex = leftEnd
			rightIndex = rightEnd
			continue
		}

		if left[leftIndex] != right[rightIndex] {
			return left[leftIndex] < right[rightIndex]
		}
		leftIndex++
		rightIndex++
	}
	if leftIndex != len(left) || rightIndex != len(right) {
		return leftIndex == len(left)
	}
	return left < right
}

func digitRunEnd(value string, start int) int {
	end := start
	for end < len(value) && isASCIIDigit(value[end]) {
		end++
	}
	return end
}

func trimLeadingZeros(value string) string {
	index := 0
	for index < len(value)-1 && value[index] == '0' {
		index++
	}
	return value[index:]
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
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
