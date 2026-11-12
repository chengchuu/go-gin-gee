import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

function run(name, ...args) {
  const cli = fileURLToPath(new URL(`../src/${name}.ts`, import.meta.url));
  const result = spawnSync(process.execPath, [ cli, ...args ], {
    encoding: "utf8", env: { ...process.env, TZ: "America/New_York" },
  });
  if (result.error) throw result.error;
  return result;
}

test("annual return preserves percentage input and output formatting", () => {
  for (const [ input, formatted ] of [ [ "20.20%", "20.20%" ], [ "20.20", "20.20%" ], [ "-15.5%", "-15.50%" ] ]) {
    const result = run("calc-annual-return", "--start", "2024-01-01", "--end", "2024-12-31", `--return=${input}`);
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout, `Investment Period: 365 days\nTotal Return     : ${formatted}\nAnnualized Return: ${formatted}\n`);
  }
  const multiYear = run("calc-annual-return", "--start", "2022-04-01", "--end", "2025-10-01", "--return=20.20%");
  assert.equal(multiYear.status, 0, multiYear.stderr);
  assert.equal(multiYear.stdout, "Investment Period: 1279 days\nTotal Return     : 20.20%\nAnnualized Return: 5.39%\n");
});

test("annual return rejects invalid dates, order, and percentages", () => {
  for (const [ start, end, rate ] of [
    [ "2024-02-30", "2024-12-31", "20%" ],
    [ "2024-12-31", "2024-01-01", "20%" ],
    [ "2024-01-01", "2024-01-01", "20%" ],
    [ "2024-01-01", "2024-12-31", "-100%" ],
    [ "2024-01-01", "2024-12-31", "20junk" ],
    [ "2024-01-01", "2024-12-31", "1e999" ],
  ]) {
    const result = run("calc-annual-return", "--start", start, "--end", end, `--return=${rate}`);
    assert.equal(result.status, 1, result.stderr);
    assert.equal(result.stdout, "");
    assert.match(result.stderr, /^Error:/);
  }
});

test("annual return uses Go fixed-point output for ties, negative zero, and large values", () => {
  for (const [ input, total, annualized ] of [
    [ "1.125%", "1.12%", "1.12%" ],
    [ "-1.125%", "-1.12%", "-1.12%" ],
    [ "-0%", "-0.00%", "0.00%" ],
    [ "1e22", "10000000000000000000000.00%", "10000000000000000000000.00%" ],
  ]) {
    const result = run("calc-annual-return", "--start", "2024-01-01", "--end", "2024-12-31", `--return=${input}`);
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout, `Investment Period: 365 days\nTotal Return     : ${total}\nAnnualized Return: ${annualized}\n`);
  }
});

test("date difference preserves flags, positional dates, reverse notice, and UTC counting", () => {
  for (const [ args, output ] of [
    [ [ "--start", "2024-03-10", "--end", "2024-03-11" ], "Days between 2024-03-10 and 2024-03-11: 1\n" ],
    [ [ "2024-02-28", "2024-03-01" ], "Days between 2024-02-28 and 2024-03-01: 2\n" ],
    [ [ "2024-03-01", "2024-03-01" ], "Days between 2024-03-01 and 2024-03-01: 0\n" ],
    [ [ "1000-01-01", "2000-01-01" ], "Days between 1000-01-01 and 2000-01-01: 365242\n" ],
    [ [ "2024-03-01", "2024-02-28" ], "Dates were provided in reverse order; computing days from 2024-03-01 to 2024-02-28.\nDays between 2024-03-01 and 2024-02-28: 2\n" ],
  ]) {
    const result = run("calc-datediff", ...args);
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout, output);
  }
  assert.equal(run("calc-datediff", "2023-02-29", "2023-03-01").status, 1);
  assert.equal(run("calc-datediff", "2024-01-01").status, 2);
});
