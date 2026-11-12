import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import ExcelJS from "exceljs";

const cli = fileURLToPath(new URL("../src/parse-excel-xlsx.ts", import.meta.url));

function fixture(t) {
  const directory = mkdtempSync(join(tmpdir(), "parse-excel-xlsx-"));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  return join(directory, "example workbook.xlsx");
}

function run(file, sheet = "fixture") {
  const result = spawnSync(process.execPath, [ cli, "--file", file, "--sheet", sheet ], { encoding: "utf8" });
  if (result.error) throw result.error;
  return result;
}

test("reads the named sheet and preserves timestamped Go log labels and counts", async t => {
  const file = fixture(t);
  const workbook = new ExcelJS.Workbook();
  workbook.addWorksheet("ignored").addRows([ [ "url" ], [ "?exampleName=Ignored" ] ]);
  const sheet = workbook.addWorksheet("fixture");
  sheet.addRows([
    [ "note", "url" ],
    [ "one", "https://example.test/?exampleName=Alpha+One&exampleId=7" ],
    [ undefined, "/relative?exampleName=Alpha%20One&exampleId=7&exampleId=8" ],
    [],
    [ "no URL" ],
    [ undefined, { text: "Display label", hyperlink: "https://example.test/?exampleId=999" } ],
    [ undefined, "?exampleName=&exampleName=Ignored&exampleId=%ZZ&exampleId=9#exampleId=10" ],
    [ undefined, "??exampleId=666" ],
    [ undefined, "?exampleId=%ZZ&?exampleId=666" ],
    [ undefined, "?exampleName=Trailing " ],
  ]);
  await workbook.xlsx.writeFile(file);
  const result = run(file);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "");
  const lines = result.stderr.trimEnd().split("\n");
  for (const line of lines) assert.match(line, /^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2} /);
  assert.deepEqual(lines.map(line => line.slice(20)), [
    "exampleName: Alpha One", "exampleId: 7", "exampleName: Alpha One", "exampleId: 7",
    "exampleId: 9", "exampleName: Trailing ", "exampleNameMap: map[Alpha One:2 Trailing :1]",
    "name: Alpha One, count: 2", "name: Trailing , count: 1",
    "exampleIdMap: map[7:2 9:1]", "id: 7, count: 2", "id: 9, count: 1",
  ]);
  const missing = run(file, "missing");
  assert.equal(missing.status, 1);
  assert.match(missing.stderr, /Worksheet 'missing' not found/);
});

test("reports missing and invalid workbook files", t => {
  const file = fixture(t);
  assert.equal(run(file).status, 1);
  writeFileSync(file, "not a workbook");
  assert.equal(run(file).status, 1);
});

test("rejects missing URL headers and reports invalid URL row numbers", async t => {
  const file = fixture(t);
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet("fixture");
  sheet.addRows([ [ "note" ], [ "https://example.test/%ZZ" ] ]);
  await workbook.xlsx.writeFile(file);
  const missing = run(file);
  assert.equal(missing.status, 1);
  assert.match(missing.stderr, /Missing 'url' column/);
  sheet.getCell("A1").value = "url";
  await workbook.xlsx.writeFile(file);
  const invalid = run(file);
  assert.equal(invalid.status, 1);
  assert.match(invalid.stderr, /Invalid URL in worksheet 'fixture', row 2/);
  sheet.getCell("A2").value = "?exampleId=7#%ZZ";
  await workbook.xlsx.writeFile(file);
  const invalidFragment = run(file);
  assert.equal(invalidFragment.status, 1);
  assert.match(invalidFragment.stderr, /Invalid URL in worksheet 'fixture', row 2/);
});
