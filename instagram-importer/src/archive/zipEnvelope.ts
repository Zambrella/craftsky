import { IMPORT_SAFETY_POLICY } from '../config/safety'

const EOCD_SIGNATURE = 0x06054b50
const ZIP64_EOCD_SIGNATURE = 0x06064b50
const ZIP64_LOCATOR_SIGNATURE = 0x07064b50
const EOCD_BYTES = 22
const MAX_COMMENT_BYTES = 65_535

export interface ZipEnvelopeLimits {
  readonly maxEntries: number
  readonly maxCentralDirectoryBytes: number
}

export interface ZipEnvelope {
  readonly zip64: boolean
  readonly entryCount: number
  readonly centralDirectoryBytes: number
  readonly centralDirectoryOffset: number
}

const DEFAULT_LIMITS: ZipEnvelopeLimits = {
  maxEntries: IMPORT_SAFETY_POLICY.maxEntries,
  maxCentralDirectoryBytes:
    IMPORT_SAFETY_POLICY.maxCentralDirectoryBytes,
}

async function readView(
  blob: Blob,
  offset: number,
  length: number,
): Promise<DataView> {
  if (
    !Number.isSafeInteger(offset) ||
    !Number.isSafeInteger(length) ||
    offset < 0 ||
    length < 0 ||
    offset + length > blob.size
  ) {
    throw new Error('invalidZipEnvelope')
  }
  return new DataView(await blob.slice(offset, offset + length).arrayBuffer())
}

function toBoundedNumber(value: bigint): number {
  if (value > BigInt(Number.MAX_SAFE_INTEGER)) {
    throw new Error('invalidZipEnvelope')
  }
  return Number(value)
}

function enforceBounds(
  envelope: ZipEnvelope,
  limits: ZipEnvelopeLimits,
): ZipEnvelope {
  if (envelope.entryCount > limits.maxEntries) {
    throw new Error('archiveEntryLimit')
  }
  if (
    envelope.centralDirectoryBytes >
    limits.maxCentralDirectoryBytes
  ) {
    throw new Error('centralDirectoryLimit')
  }
  return envelope
}

export async function inspectZipEnvelope(
  blob: Blob,
  limits: ZipEnvelopeLimits = DEFAULT_LIMITS,
): Promise<ZipEnvelope> {
  if (blob.size < EOCD_BYTES) throw new Error('invalidZipEnvelope')

  const tailOffset = Math.max(0, blob.size - EOCD_BYTES - MAX_COMMENT_BYTES)
  const tail = await readView(blob, tailOffset, blob.size - tailOffset)
  const candidates: number[] = []
  for (let offset = tail.byteLength - EOCD_BYTES; offset >= 0; offset -= 1) {
    if (tail.getUint32(offset, true) !== EOCD_SIGNATURE) continue
    const commentLength = tail.getUint16(offset + 20, true)
    if (tailOffset + offset + EOCD_BYTES + commentLength === blob.size) {
      candidates.push(tailOffset + offset)
    }
  }
  if (candidates.length !== 1) throw new Error('invalidZipEnvelope')

  const eocdOffset = candidates[0]
  const eocd = await readView(blob, eocdOffset, EOCD_BYTES)
  const disk = eocd.getUint16(4, true)
  const directoryDisk = eocd.getUint16(6, true)
  const diskEntries = eocd.getUint16(8, true)
  const totalEntries = eocd.getUint16(10, true)
  const directorySize = eocd.getUint32(12, true)
  const directoryOffset = eocd.getUint32(16, true)
  if (disk !== 0 || directoryDisk !== 0) {
    throw new Error('multiDiskZipUnsupported')
  }

  const needsZip64 =
    diskEntries === 0xffff ||
    totalEntries === 0xffff ||
    directorySize === 0xffffffff ||
    directoryOffset === 0xffffffff
  if (!needsZip64) {
    if (
      diskEntries !== totalEntries ||
      directoryOffset + directorySize > eocdOffset
    ) {
      throw new Error('invalidZipEnvelope')
    }
    return enforceBounds(
      {
        zip64: false,
        entryCount: totalEntries,
        centralDirectoryBytes: directorySize,
        centralDirectoryOffset: directoryOffset,
      },
      limits,
    )
  }

  const locatorOffset = eocdOffset - 20
  const locator = await readView(blob, locatorOffset, 20)
  if (locator.getUint32(0, true) !== ZIP64_LOCATOR_SIGNATURE) {
    throw new Error('invalidZipEnvelope')
  }
  if (locator.getUint32(4, true) !== 0 || locator.getUint32(16, true) !== 1) {
    throw new Error('multiDiskZipUnsupported')
  }
  const zip64Offset = toBoundedNumber(locator.getBigUint64(8, true))
  const zip64 = await readView(blob, zip64Offset, 56)
  if (
    zip64.getUint32(0, true) !== ZIP64_EOCD_SIGNATURE ||
    zip64.getBigUint64(4, true) < 44n
  ) {
    throw new Error('invalidZipEnvelope')
  }
  if (zip64.getUint32(16, true) !== 0 || zip64.getUint32(20, true) !== 0) {
    throw new Error('multiDiskZipUnsupported')
  }
  const entriesOnDisk = zip64.getBigUint64(24, true)
  const entries = zip64.getBigUint64(32, true)
  const zip64DirectorySize = zip64.getBigUint64(40, true)
  const zip64DirectoryOffset = zip64.getBigUint64(48, true)
  if (entriesOnDisk !== entries) throw new Error('invalidZipEnvelope')

  const result: ZipEnvelope = {
    zip64: true,
    entryCount: toBoundedNumber(entries),
    centralDirectoryBytes: toBoundedNumber(zip64DirectorySize),
    centralDirectoryOffset: toBoundedNumber(zip64DirectoryOffset),
  }
  if (
    result.centralDirectoryOffset + result.centralDirectoryBytes >
      zip64Offset ||
    zip64Offset + 56 > locatorOffset
  ) {
    throw new Error('invalidZipEnvelope')
  }
  return enforceBounds(result, limits)
}
