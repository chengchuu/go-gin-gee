import { parseArgs } from 'node:util';
import { calculateCAGR, formatPercentage, getDateDifference } from 'mazey';
import { parseUtcDate } from './utc-date.ts';

const usage = 'Usage: calc-annual-return --start YYYY-MM-DD --end YYYY-MM-DD --return=20.20%';

function formatReturn(ratio: number): string {
  const percent = ratio * 100;
  const magnitude = Math.abs(percent);
  const sign = percent < 0 || Object.is(percent, -0) ? '-' : '';
  if (!Number.isFinite(percent)) throw new Error('Formatted return must be finite.');
  if (magnitude >= 1e21) return `${sign}${BigInt(magnitude)}.00%`;
  // Go rounds exact halfway values to even; toFixed rounds them away from zero.
  const whole = Math.trunc(magnitude);
  const fraction = magnitude - whole;
  if (fraction === 0.125 || fraction === 0.625) {
    return `${sign}${whole}.${fraction === 0.125 ? '12' : '62'}%`;
  }
  if (Object.is(percent, -0)) return '-0.00%';
  return formatPercentage(ratio, 2);
}

function main() {
  const { values } = parseArgs({ options: {
    start: { type: 'string' }, end: { type: 'string' }, return: { type: 'string' },
    help: { type: 'boolean', short: 'h' },
  } });
  if (values.help) { console.log(usage); return; }
  if (!values.start || !values.end || !values.return) throw new Error(`Missing required parameters.\n${usage}`);

  const start = parseUtcDate(values.start, 'start');
  const end = parseUtcDate(values.end, 'end');
  const rate = values.return.trim().replace(/%$/, '').trim();
  if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(rate)) {
    throw new Error('Total return must be a valid number or percentage.');
  }
  const total = Number(rate) / 100;
  const annualized = calculateCAGR(start, end, total);
  const days = getDateDifference(start.getTime(), end.getTime(), { type: 'd' });
  const totalLabel = formatReturn(total);
  const annualizedLabel = formatReturn(annualized);

  console.log(`Investment Period: ${days} days`);
  console.log(`Total Return     : ${totalLabel}`);
  console.log(`Annualized Return: ${annualizedLabel}`);
}

try {
  main();
} catch (error) {
  console.error(`Error: ${(error as Error).message}`);
  process.exitCode = 1;
}
