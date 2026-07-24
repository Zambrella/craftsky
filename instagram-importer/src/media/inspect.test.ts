import { describe, expect, it } from 'vitest'

import { inspectImageHeader } from './inspect'

function png(width: number, height: number, animated = false): Uint8Array {
  const bytes = new Uint8Array(animated ? 53 : 33)
  bytes.set([137, 80, 78, 71, 13, 10, 26, 10])
  const view = new DataView(bytes.buffer)
  view.setUint32(8, 13)
  bytes.set(new TextEncoder().encode('IHDR'), 12)
  view.setUint32(16, width)
  view.setUint32(20, height)
  if (animated) {
    view.setUint32(33, 8)
    bytes.set(new TextEncoder().encode('acTL'), 37)
  }
  return bytes
}

function jpeg(width: number, height: number): Uint8Array {
  const bytes = new Uint8Array(25)
  bytes.set([0xff, 0xd8, 0xff, 0xe0, 0x00, 0x02, 0xff, 0xc0, 0x00, 0x11])
  const view = new DataView(bytes.buffer)
  view.setUint16(11, height)
  view.setUint16(13, width)
  return bytes
}

function webpExtended(
  width: number,
  height: number,
  animated = false,
): Uint8Array {
  const bytes = new Uint8Array(30)
  bytes.set(new TextEncoder().encode('RIFF'), 0)
  new DataView(bytes.buffer).setUint32(4, 22, true)
  bytes.set(new TextEncoder().encode('WEBPVP8X'), 8)
  new DataView(bytes.buffer).setUint32(16, 10, true)
  bytes[20] = animated ? 0x02 : 0
  const write24 = (offset: number, value: number) => {
    bytes[offset] = value & 0xff
    bytes[offset + 1] = (value >> 8) & 0xff
    bytes[offset + 2] = (value >> 16) & 0xff
  }
  write24(24, width - 1)
  write24(27, height - 1)
  return bytes
}

describe('header-first media inspection (UT-007)', () => {
  it('reads dimensions without decoding supported formats', () => {
    expect(inspectImageHeader(jpeg(1080, 1350))).toMatchObject({
      mime: 'image/jpeg',
      width: 1080,
      height: 1350,
    })
    expect(inspectImageHeader(png(640, 480))).toMatchObject({
      mime: 'image/png',
      width: 640,
      height: 480,
    })
    expect(inspectImageHeader(webpExtended(320, 240))).toMatchObject({
      mime: 'image/webp',
      width: 320,
      height: 240,
    })
  })

  it('rejects animation and limits before browser decode', () => {
    expect(() => inspectImageHeader(png(20, 20, true))).toThrow(
      'animatedImageUnsupported',
    )
    expect(() => inspectImageHeader(webpExtended(20, 20, true))).toThrow(
      'animatedImageUnsupported',
    )
    expect(() => inspectImageHeader(png(12_001, 1))).toThrow(
      'imageDimensionLimit',
    )
    expect(() => inspectImageHeader(png(6_000, 5_000))).toThrow(
      'imagePixelLimit',
    )
  })

  it('keeps dimension and decoded-pixel limits inclusive', () => {
    expect(inspectImageHeader(png(12_000, 1))).toMatchObject({
      width: 12_000,
      height: 1,
    })
    expect(() => inspectImageHeader(png(12_001, 1))).toThrow(
      'imageDimensionLimit',
    )
    expect(inspectImageHeader(png(5_000, 5_000))).toMatchObject({
      width: 5_000,
      height: 5_000,
    })
    expect(() => inspectImageHeader(png(5_000, 5_001))).toThrow(
      'imagePixelLimit',
    )
  })

  it('rejects corrupt or unsupported signatures', () => {
    expect(() => inspectImageHeader(new Uint8Array([0, 1, 2]))).toThrow(
      'unsupportedImageFormat',
    )
  })
})
