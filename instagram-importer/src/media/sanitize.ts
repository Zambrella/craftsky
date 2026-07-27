import { IMPORT_SAFETY_POLICY } from '../config/safety'
import {
  inspectImageHeader,
  type SupportedImageMime,
} from './inspect'

export interface SanitizedImage {
  readonly buffer: ArrayBuffer
  readonly mime: SupportedImageMime
  readonly width: number
  readonly height: number
}

export async function sanitizeImage(
  source: Uint8Array,
  signal?: AbortSignal,
): Promise<SanitizedImage> {
  signal?.throwIfAborted()
  if (source.byteLength > IMPORT_SAFETY_POLICY.maxSourceImageBytes) {
    throw new Error('sourceImageLimit')
  }
  const header = inspectImageHeader(source)
  if (
    typeof OffscreenCanvas === 'undefined' ||
    typeof createImageBitmap === 'undefined'
  ) {
    throw new Error('browserMediaUnsupported')
  }

  const sourceBuffer = source.slice().buffer
  const sourceBlob = new Blob([sourceBuffer], { type: header.mime })
  const bitmap = await createImageBitmap(sourceBlob)
  try {
    signal?.throwIfAborted()
    if (
      bitmap.width < 1 ||
      bitmap.height < 1 ||
      bitmap.width > IMPORT_SAFETY_POLICY.maxDimension ||
      bitmap.height > IMPORT_SAFETY_POLICY.maxDimension ||
      bitmap.width * bitmap.height >
        IMPORT_SAFETY_POLICY.maxDecodedPixels
    ) {
      throw new Error('decodedImageLimit')
    }
    let width = bitmap.width
    let height = bitmap.height
    for (let attempt = 0; attempt < 8; attempt += 1) {
      signal?.throwIfAborted()
      const canvas = new OffscreenCanvas(width, height)
      const context = canvas.getContext('2d', { alpha: true })
      if (!context) throw new Error('browserMediaUnsupported')
      context.drawImage(bitmap, 0, 0, width, height)
      const encoded = await canvas.convertToBlob({
        type: header.mime,
        quality: header.mime === 'image/png' ? undefined : 0.9,
      })
      signal?.throwIfAborted()
      const buffer = await encoded.arrayBuffer()
      signal?.throwIfAborted()
      if (buffer.byteLength <= IMPORT_SAFETY_POLICY.maxFinalBlobBytes) {
        const outputHeader = inspectImageHeader(new Uint8Array(buffer))
        if (outputHeader.mime !== header.mime) {
          throw new Error('mediaEncodingMismatch')
        }
        return {
          buffer,
          mime: outputHeader.mime,
          width: outputHeader.width,
          height: outputHeader.height,
        }
      }
      width = Math.max(1, Math.floor(width * 0.82))
      height = Math.max(1, Math.floor(height * 0.82))
    }
    throw new Error('finalImageLimit')
  } finally {
    bitmap.close()
  }
}
