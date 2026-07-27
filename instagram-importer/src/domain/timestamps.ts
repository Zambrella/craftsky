const INSTAGRAM_LAUNCH_MS = Date.parse('2010-10-06T00:00:00.000Z')
const FUTURE_ALLOWANCE_MS = 24 * 60 * 60 * 1000

export function canonicalizeSourceTimestamp(
  value: unknown,
): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value < 100_000_000_000 ? value * 1000 : value
  }
  if (typeof value === 'string' && value.trim()) {
    const parsed = Date.parse(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

export function normalizeSourceTimestamp(
  value: unknown,
  now = new Date(),
): string | null {
  const milliseconds = canonicalizeSourceTimestamp(value)
  if (
    milliseconds === null ||
    milliseconds < INSTAGRAM_LAUNCH_MS ||
    milliseconds > now.getTime() + FUTURE_ALLOWANCE_MS
  ) {
    return null
  }
  return new Date(milliseconds).toISOString()
}
