package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const isoLayout = "2006-01-02"

func parseDateToUTC(s string) (time.Time, error) {
	t, err := time.Parse(isoLayout, s)
	if err != nil {
		return time.Time{}, err
	}
	// Normalize to midnight UTC to avoid timezone/DST issues
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s -start YYYY-MM-DD -end YYYY-MM-DD\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "Or:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s YYYY-MM-DD YYYY-MM-DD\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "\nFlags:\n")
	flag.PrintDefaults()
}

func main() {
	startPtr := flag.String("start", "", "start date in YYYY-MM-DD")
	endPtr := flag.String("end", "", "end date in YYYY-MM-DD")
	flag.Usage = usage
	flag.Parse()

	// Allow positional args if -start/-end not provided:
	args := flag.Args()
	if *startPtr == "" && *endPtr == "" && len(args) >= 2 {
		*startPtr = args[0]
		*endPtr = args[1]
	}

	if *startPtr == "" || *endPtr == "" {
		usage()
		os.Exit(2)
	}

	d1, err := parseDateToUTC(*startPtr)
	if err != nil {
		log.Fatalf("failed to parse start date %q: %v", *startPtr, err)
	}
	d2, err := parseDateToUTC(*endPtr)
	if err != nil {
		log.Fatalf("failed to parse end date %q: %v", *endPtr, err)
	}

	swapped := false
	if d2.Before(d1) {
		d1, d2 = d2, d1
		swapped = true
	}

	// Both times are normalized to midnight UTC, so difference is a whole number of days.
	diff := d2.Sub(d1)
	days := int(diff.Hours() / 24)

	if swapped {
		fmt.Printf("Dates were provided in reverse order; computing days from %s to %s.\n", *startPtr, *endPtr)
	}
	fmt.Printf("Days between %s and %s: %d\n", *startPtr, *endPtr, days)
}
