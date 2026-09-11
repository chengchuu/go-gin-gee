# Investment Annualized Return Calculator

A lightweight, production-ready Go command-line tool designed to calculate the compound annual growth rate (CAGR) for financial investments. Given a start date, end date, and total return rate, the tool computes the precise investment duration in days and outputs the annualized return.

- [Formula](#formula)
- [Features](#features)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Project Structure](#project-structure)
- [Usage](#usage)
  - [CLI Arguments](#cli-arguments)
- [Examples](#examples)
  - [Positive Return](#positive-return)
  - [Negative Return](#negative-return)
- [Error Handling](#error-handling)

## Formula

The tool uses the standard Compound Annual Growth Rate (CAGR) formula:

$$\text{Annualized Return} = (1 + \text{Total Return})^{\frac{365}{\text{Days}}} - 1$$

*Note: Total Return is automatically converted from a percentage representation to a decimal for calculation.*

---

## Features

- **Flexible Input Formats:** Accepts total return rates both with or without a percentage sign (e.g., `20.20%` or `20.20`).
- **Robust Validation:** - Validates date sequencing (ensures the end date is strictly after the start date).
  - Handles negative returns appropriately.
  - Rejects impossible financial scenarios (e.g., total capital wipeout of $-100\%$ or less).
- **Zero Dependencies:** Relies entirely on the Go standard library (`flag`, `time`, `math`).

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/doc/install) (version 1.16 or higher recommended)

### Project Structure

Ensure your project directory structure matches the following layout:
```text
.
└── scripts/
    └── calc-annual-return/
        └── main.go
```

---

## Usage

You can run the script directly using `go run`:

```bash
go run scripts/calc-annual-return/main.go -start <YYYY-MM-DD> -end <YYYY-MM-DD> -return <value>
```

### CLI Arguments

| Flag | Description | Format | Required |
| --- | --- | --- | --- |
| `-start` | The date the investment was made | `YYYY-MM-DD` | **Yes** |
| `-end` | The date the investment was matured/evaluated | `YYYY-MM-DD` | **Yes** |
| `-return` | The total return rate over the entire period | `20.20%` or `20.20` | **Yes** |

---

## Examples

### Positive Return

```bash
go run scripts/calc-annual-return/main.go -start 2022-04-01 -end 2025-10-01 -return 20.20%
```

**Output:**

```text
Investment Period: 1279 days
Total Return:      20.20%
Annualized Return: 5.41%
```

### Negative Return

```bash
go run scripts/calc-annual-return/main.go -start 2022-04-01 -end 2025-10-01 -return -15.5%
```

**Output:**

```text
Investment Period: 1279 days
Total Return:      -15.50%
Annualized Return: -4.73%
```

---

## Error Handling

The application gracefully handles common user errors and outputs clear messages to `stderr`:

* **Missing Parameters:** `Error: Missing required parameters.`
* **Invalid Date Format:** `Error: Invalid start date format '04-01-2022'. Use YYYY-MM-DD.`
* **Chronological Order:** `Error: End date must be strictly after the start date.`
* **Extreme Losses:** `Error: Total return cannot be -100% or less (total capital loss or worse).`
