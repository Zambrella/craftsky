const MEBIBYTE = 1024 * 1024

export const IMPORT_SAFETY_POLICY = Object.freeze({
  id: 'instagram-import-v1' as const,
  maxEntries: 100_000,
  maxCentralDirectoryBytes: 64 * MEBIBYTE,
  maxCandidateJsonBytes: 32 * MEBIBYTE,
  maxCombinedCandidateJsonBytes: 128 * MEBIBYTE,
  maxPosts: 25_000,
  maxSourceImageBytes: 64 * MEBIBYTE,
  maxDecodedPixels: 25_000_000,
  maxDimension: 12_000,
  maxCompressionRatio: 200,
  maxConcurrentImageTransforms: 1,
  maxFinalBlobBytes: 15 * MEBIBYTE,
})

export function withinInclusiveLimit(value: number, limit: number): boolean {
  return Number.isFinite(value) && value >= 0 && value <= limit
}

export function withinCompressionRatio(
  uncompressedSize: number,
  compressedSize: number,
  maxRatio: number,
): boolean {
  if (
    !Number.isFinite(uncompressedSize) ||
    !Number.isFinite(compressedSize) ||
    !Number.isFinite(maxRatio) ||
    uncompressedSize < 0 ||
    compressedSize < 0 ||
    maxRatio < 0
  ) {
    return false
  }
  if (uncompressedSize === 0) return true
  return (
    compressedSize > 0 &&
    uncompressedSize / compressedSize <= maxRatio
  )
}
