package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// go run scripts/calc-annual-return/main.go -start 2022-04-01 -end 2025-10-01 -return 20.20%
func main() {
	// 1. Define CLI flags
	startStr := flag.String("start", "", "Investment start date (YYYY-MM-DD)")
	endStr := flag.String("end", "", "Investment end date (YYYY-MM-DD)")
	returnStr := flag.String("return", "", "Total return rate (e.g., 20.20% or 20.20)")

	flag.Parse()

	// 2. Validate missing parameters
	if *startStr == "" || *endStr == "" || *returnStr == "" {
		fmt.Fprintln(os.Stderr, "Error: Missing required parameters.")
		flag.Usage()
		os.Exit(1)
	}

	// 3. Parse and validate dates
	dateFormat := "2006-01-02"
	startDate, err := time.Parse(dateFormat, *startStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid start date format '%s'. Use YYYY-MM-DD.\n", *startStr)
		os.Exit(1)
	}

	endDate, err := time.Parse(dateFormat, *endStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid end date format '%s'. Use YYYY-MM-DD.\n", *endStr)
		os.Exit(1)
	}

	if !endDate.After(startDate) {
		fmt.Fprintln(os.Stderr, "Error: End date must be strictly after the start date.")
		os.Exit(1)
	}

	// 4. Parse and validate the return rate
	totalReturn, err := parseReturnRate(*returnStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid return rate format '%s'. %v\n", *returnStr, err)
		os.Exit(1)
	}

	// Total return cannot be -100% or less, as a 100% loss means total wipeout (CAGR becomes undefined/impossible)
	if totalReturn <= -1.0 {
		fmt.Fprintln(os.Stderr, "Error: Total return cannot be -100% or less (total capital loss or worse).")
		os.Exit(1)
	}

	// 5. Calculate investment duration in days
	duration := endDate.Sub(startDate)
	days := duration.Hours() / 24.0

	// Edge case: Safety check for zero days (though endDate.After guards against this)
	if days <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Investment duration must be at least 1 day.")
		os.Exit(1)
	}

	// 6. Calculate Annualized Return (CAGR)
	// Formula: (1 + Total Return)^(365 / Days) - 1
	annualizedReturn := math.Pow(1.0+totalReturn, 365.0/days) - 1.0

	// 7. Output Results
	fmt.Printf("Investment Period: %.0f days\n", days)
	fmt.Printf("Total Return     : %.2f%%\n", totalReturn*100)
	fmt.Printf("Annualized Return: %.2f%%\n", annualizedReturn*100)
}

// parseReturnRate handles both "20.20%" and "20.20" formats and converts them to a decimal (0.202)
func parseReturnRate(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")

	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return 0, fmt.Errorf("must be a valid number or percentage")
	}

	// Convert percentage to decimal representation
	return val / 100.0, nil
}
