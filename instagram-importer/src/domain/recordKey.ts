import { TID } from '@atproto/common-web'

export interface DeterministicRecordKeySource {
  readonly createdAt: string
  readonly canonicalIdentityKey: string
}

/**
 * Maps canonical source identities to deterministic TIDs.
 *
 * Each source's 128-bit canonical identity digest selects one of the TID slots
 * in its source UTC day. That keeps the TID timestamp valid and close to the
 * source date while making the result independent of the other posts present
 * in an export. A digest/slot collision deliberately remains a duplicate rkey
 * so the collision safety boundary can fail closed.
 */
export function deterministicRecordKeys(
  sources: readonly DeterministicRecordKeySource[],
): string[] {
  const millisecondsPerDay = 86_400_000
  const microsecondsPerDay = 86_400_000_000n
  const clockIdsPerMicrosecond = 1_024n
  const slotsPerDay = microsecondsPerDay * clockIdsPerMicrosecond
  return sources.map((source) => {
    const milliseconds = Date.parse(source.createdAt)
    if (!Number.isFinite(milliseconds)) {
      throw new Error('invalidCreatedAt')
    }
    if (!/^[0-9a-f]{32}$/.test(source.canonicalIdentityKey)) {
      throw new Error('invalidCanonicalIdentityKey')
    }
    const dayStartMilliseconds =
      Math.floor(milliseconds / millisecondsPerDay) * millisecondsPerDay
    const slot =
      BigInt(`0x${source.canonicalIdentityKey}`) % slotsPerDay
    const timestampMicros =
      BigInt(dayStartMilliseconds) * 1_000n +
      slot / clockIdsPerMicrosecond
    const clockId = slot % clockIdsPerMicrosecond
    return TID.fromTime(
      Number(timestampMicros),
      Number(clockId),
    ).toString()
  })
}
