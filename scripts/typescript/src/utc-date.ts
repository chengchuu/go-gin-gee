export function parseUtcDate(value: string, label: string): Date {
  const date = new Date(`${value}T00:00:00.000Z`);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || !Number.isFinite(date.getTime()) ||
      date.toISOString().slice(0, 10) !== value) {
    throw new Error(`Invalid ${label} date '${value}'. Use YYYY-MM-DD.`);
  }
  return date;
}
