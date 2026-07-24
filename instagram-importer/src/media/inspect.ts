import { IMPORT_SAFETY_POLICY } from '../config/safety'

export type SupportedImageMime =
  | 'image/jpeg'
  | 'image/png'
  | 'image/webp'

export interface ImageHeader {
  readonly mime: SupportedImageMime
  readonly width: number
  readonly height: number
  readonly animated: false
}

const ASCII = new TextDecoder('ascii')

function ascii(bytes: Uint8Array, start: number, length: number): string {
  return ASCII.decode(bytes.subarray(start, start + length))
}

function validateDimensions(
  mime: SupportedImageMime,
  width: number,
  height: number,
): ImageHeader {
  if (
    !Number.isInteger(width) ||
    !Number.isInteger(height) ||
    width < 1 ||
    height < 1
  ) {
    throw new Error('corruptImageHeader')
  }
  if (
    width > IMPORT_SAFETY_POLICY.maxDimension ||
    height > IMPORT_SAFETY_POLICY.maxDimension
  ) {
    throw new Error('imageDimensionLimit')
  }
  if (width * height > IMPORT_SAFETY_POLICY.maxDecodedPixels) {
    throw new Error('imagePixelLimit')
  }
  return { mime, width, height, animated: false }
}

function inspectJpeg(bytes: Uint8Array): ImageHeader | null {
  if (bytes.length < 4 || bytes[0] !== 0xff || bytes[1] !== 0xd8) {
    return null
  }
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  let offset = 2
  while (offset + 4 <= bytes.length) {
    if (bytes[offset] !== 0xff) throw new Error('corruptImageHeader')
    while (offset < bytes.length && bytes[offset] === 0xff) offset += 1
    const marker = bytes[offset]
    offset += 1
    if (marker === undefined) break
    if (marker === 0xd9 || marker === 0xda) break
    if (marker >= 0xd0 && marker <= 0xd7) continue
    if (offset + 2 > bytes.length) throw new Error('corruptImageHeader')
    const length = view.getUint16(offset)
    if (length < 2 || offset + length > bytes.length) {
      throw new Error('corruptImageHeader')
    }
    const isStartOfFrame =
      (marker >= 0xc0 && marker <= 0xc3) ||
      (marker >= 0xc5 && marker <= 0xc7) ||
      (marker >= 0xc9 && marker <= 0xcb) ||
      (marker >= 0xcd && marker <= 0xcf)
    if (isStartOfFrame) {
      if (length < 7) throw new Error('corruptImageHeader')
      return validateDimensions(
        'image/jpeg',
        view.getUint16(offset + 5),
        view.getUint16(offset + 3),
      )
    }
    offset += length
  }
  throw new Error('corruptImageHeader')
}

function inspectPng(bytes: Uint8Array): ImageHeader | null {
  const signature = [137, 80, 78, 71, 13, 10, 26, 10]
  if (!signature.every((value, index) => bytes[index] === value)) return null
  if (bytes.length < 33 || ascii(bytes, 12, 4) !== 'IHDR') {
    throw new Error('corruptImageHeader')
  }
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  const width = view.getUint32(16)
  const height = view.getUint32(20)
  let offset = 8
  while (offset + 12 <= bytes.length) {
    const length = view.getUint32(offset)
    const type = ascii(bytes, offset + 4, 4)
    if (type === 'acTL') throw new Error('animatedImageUnsupported')
    if (type === 'IDAT' || type === 'IEND') break
    const next = offset + 12 + length
    if (next > bytes.length) break
    offset = next
  }
  return validateDimensions('image/png', width, height)
}

function readUint24(bytes: Uint8Array, offset: number): number {
  return (
    (bytes[offset] ?? 0) |
    ((bytes[offset + 1] ?? 0) << 8) |
    ((bytes[offset + 2] ?? 0) << 16)
  )
}

function inspectWebp(bytes: Uint8Array): ImageHeader | null {
  if (
    bytes.length < 20 ||
    ascii(bytes, 0, 4) !== 'RIFF' ||
    ascii(bytes, 8, 4) !== 'WEBP'
  ) {
    return null
  }
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  let offset = 12
  let dimensions: { width: number; height: number } | null = null
  while (offset + 8 <= bytes.length) {
    const type = ascii(bytes, offset, 4)
    const length = view.getUint32(offset + 4, true)
    const dataOffset = offset + 8
    if (dataOffset + length > bytes.length) {
      throw new Error('corruptImageHeader')
    }
    if (type === 'ANIM' || type === 'ANMF') {
      throw new Error('animatedImageUnsupported')
    }
    if (type === 'VP8X') {
      if (length < 10) throw new Error('corruptImageHeader')
      if ((bytes[dataOffset] ?? 0) & 0x02) {
        throw new Error('animatedImageUnsupported')
      }
      dimensions = {
        width: readUint24(bytes, dataOffset + 4) + 1,
        height: readUint24(bytes, dataOffset + 7) + 1,
      }
    } else if (type === 'VP8L' && !dimensions) {
      if (length < 5 || bytes[dataOffset] !== 0x2f) {
        throw new Error('corruptImageHeader')
      }
      const bits = view.getUint32(dataOffset + 1, true)
      dimensions = {
        width: (bits & 0x3fff) + 1,
        height: ((bits >>> 14) & 0x3fff) + 1,
      }
    } else if (type === 'VP8 ' && !dimensions) {
      if (
        length < 10 ||
        bytes[dataOffset + 3] !== 0x9d ||
        bytes[dataOffset + 4] !== 0x01 ||
        bytes[dataOffset + 5] !== 0x2a
      ) {
        throw new Error('corruptImageHeader')
      }
      dimensions = {
        width: view.getUint16(dataOffset + 6, true) & 0x3fff,
        height: view.getUint16(dataOffset + 8, true) & 0x3fff,
      }
    }
    offset = dataOffset + length + (length % 2)
  }
  if (!dimensions) throw new Error('corruptImageHeader')
  return validateDimensions('image/webp', dimensions.width, dimensions.height)
}

export function inspectImageHeader(bytes: Uint8Array): ImageHeader {
  const inspected =
    inspectJpeg(bytes) ?? inspectPng(bytes) ?? inspectWebp(bytes)
  if (!inspected) throw new Error('unsupportedImageFormat')
  return inspected
}
