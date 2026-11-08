import { parseArgs } from 'node:util';
import { getDateDifference } from 'mazey';
import { parseUtcDate } from './utc-date.ts';

const usage = 'Usage: calc-datediff --start YYYY-MM-DD --end YYYY-MM-DD\n' +
  '   or: calc-datediff YYYY-MM-DD YYYY-MM-DD';

function main() {
  let start: string;
  let end: string;
  try {
    const { values, positionals } = parseArgs({ allowPositionals: true, options: {
      start: { type: 'string' }, end: { type: 'string' }, help: { type: 'boolean', short: 'h' },
    } });
    if (values.help) { console.log(usage); return; }
    if (values.start && values.end && positionals.length === 0) {
      start = values.start;
      end = values.end;
    } else if (values.start === undefined && values.end === undefined && positionals.length === 2) {
      [start, end] = positionals;
    } else {
      throw new Error('Provide two dates using either flags or positional arguments.');
    }
  } catch (error) {
    console.error(`${(error as Error).message}\n${usage}`);
    process.exitCode = 2;
    return;
  }

  const first = parseUtcDate(start, 'start').getTime();
  const second = parseUtcDate(end, 'end').getTime();
  const days = getDateDifference(Math.min(first, second), Math.max(first, second), { type: 'd' });
  if (second < first) {
    console.log(`Dates were provided in reverse order; computing days from ${start} to ${end}.`);
  }
  console.log(`Days between ${start} and ${end}: ${days}`);
}

try {
  main();
} catch (error) {
  console.error(`Error: ${(error as Error).message}`);
  process.exitCode = 1;
}
