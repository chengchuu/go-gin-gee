# datediff

A small Go command-line utility to calculate the number of days between two dates.

- [datediff](#datediff)
  - [Usage](#usage)
  - [Behavior and notes](#behavior-and-notes)
  - [Examples (copy/paste)](#examples-copypaste)
  - [Contributing](#contributing)

## Usage

Dates must be in ISO format: `YYYY-MM-DD`.

Examples:

- Positional:
  - `go run scripts/datediff/main.go 2022-04-01 2025-10-01`
  - Output: `Days between 2022-04-01 and 2025-10-01: 1264`

- Flags:
  - `go run scripts/datediff/main.go -start 2022-04-01 -end 2025-10-01`
  - Output: `Days between 2022-04-01 and 2025-10-01: 1264`

- Reverse order (the tool will swap and notify):
  - `go run scripts/datediff/main.go 2025-10-01 2022-04-01`
  - Output:
    - `Dates were provided in reverse order; computing days from 2025-10-01 to 2022-04-01.`
    - `Days between 2025-10-01 and 2022-04-01: 1264`

## Behavior and notes

- The program normalizes both parsed dates to midnight UTC (00:00:00 UTC) to avoid timezone and DST issues. Because of that, the difference is always a whole number of days.
- The result is exclusive: it reports the number of full days between the two dates (it does not add 1 for inclusive counting).
- If either date cannot be parsed, the program exits with an error message.
- If required flags/arguments are missing, the program prints usage and exits with status code 2.

## Examples (copy/paste)

- `go run scripts/datediff/main.go` 2022-04-01 2025-10-01
- `go run scripts/datediff/main.go` -start 2022-04-01 -end 2025-10-01

## Contributing

Small changes welcome: open PRs to improve usage messages, add tests, or add an optional inclusive mode if you need inclusive counting.
