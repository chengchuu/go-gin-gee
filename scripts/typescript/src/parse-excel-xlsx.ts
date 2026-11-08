import { parseArgs } from 'node:util';
import ExcelJS from 'exceljs';

const usage = 'Usage: parse-excel-xlsx --file PATH --sheet NAME';
const invalidEscape = /%(?![0-9a-f]{2})/i;

function log(message: string) {
  const date = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  const day = `${date.getFullYear()}/${pad(date.getMonth() + 1)}/${pad(date.getDate())}`;
  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
  console.error(`${day} ${time} ${message}`);
}

function queryValues(text: string): URLSearchParams {
  const hash = text.indexOf('#');
  const reference = hash < 0 ? text : text.slice(0, hash);
  const queryStart = reference.indexOf('?');
  const path = queryStart < 0 ? reference : reference.slice(0, queryStart);
  if (/[\x00-\x1f\x7f]/.test(text) || invalidEscape.test(path) ||
      (hash >= 0 && invalidEscape.test(text.slice(hash + 1)))) throw new Error('Invalid URL text.');
  new URL(text, 'https://example.invalid/');
  const rawQuery = queryStart < 0 ? '' : reference.slice(queryStart + 1);
  // Go's URL.Query skips malformed pairs and unescaped semicolons.
  const pairs = rawQuery.split('&').filter(pair => !pair.includes(';') && !invalidEscape.test(pair));
  return new URLSearchParams(`?${pairs.join('&')}`);
}

function record(query: URLSearchParams, key: string, counts: Map<string, number>) {
  const value = query.get(key);
  if (!value) return;
  counts.set(value, (counts.get(value) ?? 0) + 1);
  log(`${key}: ${value}`);
}

function report(key: string, label: string, counts: Map<string, number>) {
  const keys = [...counts.keys()].sort();
  log(`${key}Map: map[${keys.map(value => `${value}:${counts.get(value)}`).join(' ')}]`);
  for (const value of keys) log(`${label}: ${value}, count: ${counts.get(value)}`);
}

async function main() {
  const { values } = parseArgs({ options: {
    file: { type: 'string' }, sheet: { type: 'string' }, help: { type: 'boolean', short: 'h' },
  } });
  if (values.help) { console.log(usage); return; }
  if (!values.file || !values.sheet) throw new Error(`--file and --sheet are required.\n${usage}`);

  const workbook = new ExcelJS.Workbook();
  await workbook.xlsx.readFile(values.file);
  const sheet = workbook.getWorksheet(values.sheet);
  if (!sheet) throw new Error(`Worksheet '${values.sheet}' not found.`);

  const names = new Map<string, number>();
  const ids = new Map<string, number>();
  let headerRead = false;
  let column = 0;
  sheet.eachRow(row => {
    if (!headerRead) {
      headerRead = true;
      row.eachCell((cell, index) => {
        if (cell.text === 'url') {
          if (column) throw new Error("Duplicate 'url' column.");
          column = index;
        }
      });
      if (!column) throw new Error("Missing 'url' column.");
      return;
    }
    const text = row.getCell(column).text;
    if (!text) return;
    let query: URLSearchParams;
    try {
      query = queryValues(text);
    } catch (error) {
      throw new Error(`Invalid URL in worksheet '${sheet.name}', row ${row.number}.`, { cause: error });
    }
    record(query, 'exampleName', names);
    record(query, 'exampleId', ids);
  });
  if (!headerRead) throw new Error("Missing 'url' column.");
  report('exampleName', 'name', names);
  report('exampleId', 'id', ids);
}

try {
  await main();
} catch (error) {
  console.error(`Error: ${(error as Error).message}`);
  process.exitCode = 1;
}
