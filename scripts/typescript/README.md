# TypeScript Tools

- [Fetch Complete Git Branch History](#fetch-complete-git-branch-history)
- [Calculate Duolingo Mutual Follows](#calculate-duolingo-mutual-follows)
- [Calculate Annualized Return and Elapsed Days](#calculate-annualized-return-and-elapsed-days)
- [Count URL Parameters in an Excel Worksheet](#count-url-parameters-in-an-excel-worksheet)

## Fetch Complete Git Branch History

Use the TypeScript CLI to download Git objects omitted by a partial clone.
Install Node.js 22.18.0 or later and Git 2.36 or later. From the repository root,
run the source file directly:

```bash
node scripts/typescript/src/fetch-complete-branches.ts \
  --path "/Users/cheng/third/react-scan" --batch-size 500
```

The CLI refreshes all branch refs from the existing `origin` and downloads
missing objects in sequential batches. Local branches and working files are
preserved. Use a smaller `--batch-size` if transfers keep failing. This setting
limits object count, so a batch containing a large blob can still take time.
Use `--retries` to change the default of three retries for recognized network
failures. Permanent failures stop the run.

Press Ctrl+C to stop. Rerun the same command to resume from objects Git has
already stored. Run one downloader at a time for each clone and avoid other Git
operations while it runs. The CLI removes origin's partial-clone settings only
after verifying that all objects reachable from local refs are present. If origin
cannot supply an object referenced only by local history or another remote, the
CLI stops and preserves the partial-clone settings.

Git LFS payloads and submodule repositories are outside this command's scope.

Node.js runs the Git CLI without a build or npm dependencies. To check types without
emitting JavaScript, install the development dependencies and run the TypeScript
7 check:

```bash
npm --prefix scripts/typescript ci
npm --prefix scripts/typescript run typecheck
```

Run the integration tests with `npm --prefix scripts/typescript test`. The tests
run the same source file against temporary repositories.

## Calculate Duolingo Mutual Follows

Run the source file directly with Node.js 22.18.0 or later. Pass both JSON exports
from the repository root:

```bash
node scripts/typescript/src/calc-mutuals.ts \
  --following scripts/typescript/data/duolingo/following_20251111.json \
  --followers scripts/typescript/data/duolingo/followers_20251111.json
```

Each export must contain an array of records with a numeric `userId` and string
`username` and `displayName` fields. The command compares IDs and lists people
you follow who do not follow you back, in the order of the following export.

## Calculate Annualized Return and Elapsed Days

Install the Mazey runtime dependency from the lockfile, then run either calculator
directly from the repository root:

```bash
npm --prefix scripts/typescript ci
node scripts/typescript/src/calc-annual-return.ts --start 2022-04-01 --end 2025-10-01 --return=20.20%
node scripts/typescript/src/calc-datediff.ts --start 2022-04-01 --end 2025-10-01
node scripts/typescript/src/calc-datediff.ts 2022-04-01 2025-10-01
```

Both commands interpret dates as midnight UTC. The annualized return calculator
uses a 365-day year and accepts a percentage with or without `%`. Use `=` for a
negative return, for example `--return=-15.5%`. The end date must be after the
start date, and the total return must be greater than `-100%`.

The date difference calculator counts elapsed days without adding an inclusive
day. It reports zero for equal dates and prints a notice for reversed dates.

## Count URL Parameters in an Excel Worksheet

Install dependencies with `npm --prefix scripts/typescript ci`, then run:

```bash
node scripts/typescript/src/parse-excel-xlsx.ts --file data/example_name.xlsx --sheet example_name
```

The first populated row must contain a `url` header. The CLI reads the cell text
and logs nonempty `exampleName` and `exampleId` query values and their counts to
stderr. Blank URL cells are skipped. Hyperlink targets are not followed and
formulas are not evaluated. The workbook is read into memory and is not modified.
