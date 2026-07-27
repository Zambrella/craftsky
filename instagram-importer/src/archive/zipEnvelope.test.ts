import { describe, expect, it } from 'vitest'

import { inspectZipEnvelope } from './zipEnvelope'

function classicZip(
  entryCount: number,
  directorySize: number,
  directoryOffset = 0,
): Blob {
  const directory = new Uint8Array(directorySize)
  const eocd = new ArrayBuffer(22)
  const view = new DataView(eocd)
  view.setUint32(0, 0x06054b50, true)
  view.setUint16(8, entryCount, true)
  view.setUint16(10, entryCount, true)
  view.setUint32(12, directorySize, true)
  view.setUint32(16, directoryOffset, true)
  return new Blob([directory, eocd])
}

function zip64(entryCount: bigint, directorySize: bigint): Blob {
  const directory = new Uint8Array(Number(directorySize))
  const record = new ArrayBuffer(56)
  const recordView = new DataView(record)
  recordView.setUint32(0, 0x06064b50, true)
  recordView.setBigUint64(4, 44n, true)
  recordView.setUint16(12, 45, true)
  recordView.setUint16(14, 45, true)
  recordView.setBigUint64(24, entryCount, true)
  recordView.setBigUint64(32, entryCount, true)
  recordView.setBigUint64(40, directorySize, true)
  recordView.setBigUint64(48, 0n, true)

  const locator = new ArrayBuffer(20)
  const locatorView = new DataView(locator)
  locatorView.setUint32(0, 0x07064b50, true)
  locatorView.setBigUint64(8, BigInt(directory.byteLength), true)
  locatorView.setUint32(16, 1, true)

  const eocd = new ArrayBuffer(22)
  const eocdView = new DataView(eocd)
  eocdView.setUint32(0, 0x06054b50, true)
  eocdView.setUint16(8, 0xffff, true)
  eocdView.setUint16(10, 0xffff, true)
  eocdView.setUint32(12, 0xffffffff, true)
  eocdView.setUint32(16, 0xffffffff, true)
  return new Blob([directory, record, locator, eocd])
}

describe('ZIP envelope preflight (IT-002)', () => {
  it('reads classic and ZIP64 metadata before zip.js', async () => {
    await expect(inspectZipEnvelope(classicZip(3, 8))).resolves.toEqual({
      zip64: false,
      entryCount: 3,
      centralDirectoryBytes: 8,
      centralDirectoryOffset: 0,
    })
    await expect(inspectZipEnvelope(zip64(4n, 12n))).resolves.toEqual({
      zip64: true,
      entryCount: 4,
      centralDirectoryBytes: 12,
      centralDirectoryOffset: 0,
    })
  })

  it('rejects the first value above each inclusive bound', async () => {
    await expect(
      inspectZipEnvelope(classicZip(11, 8), {
        maxEntries: 10,
        maxCentralDirectoryBytes: 8,
      }),
    ).rejects.toThrow('archiveEntryLimit')
    await expect(
      inspectZipEnvelope(classicZip(2, 9), {
        maxEntries: 10,
        maxCentralDirectoryBytes: 8,
      }),
    ).rejects.toThrow('centralDirectoryLimit')
  })

  it('rejects malformed offsets and multi-disk archives', async () => {
    const invalidOffset = classicZip(1, 8, 9)
    await expect(inspectZipEnvelope(invalidOffset)).rejects.toThrow(
      'invalidZipEnvelope',
    )

    const bytes = new Uint8Array(await classicZip(1, 8).arrayBuffer())
    new DataView(bytes.buffer).setUint16(bytes.byteLength - 22 + 4, 1, true)
    await expect(inspectZipEnvelope(new Blob([bytes]))).rejects.toThrow(
      'multiDiskZipUnsupported',
    )
  })
})
