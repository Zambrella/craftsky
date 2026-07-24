import { describe, expect, it, vi } from 'vitest'

import type { ReviewManifest } from '../domain/types'
import { ArchiveWorkerClient, type WorkerPort } from './client'

class FakeWorker implements WorkerPort {
  readonly sent: unknown[] = []
  private listener?: (event: MessageEvent) => void

  postMessage(message: unknown): void {
    this.sent.push(message)
  }

  addEventListener(
    _type: 'message',
    listener: (event: MessageEvent) => void,
  ): void {
    this.listener = listener
  }

  removeEventListener(): void {}

  emit(data: unknown): void {
    this.listener?.(new MessageEvent('message', { data }))
  }
}

const manifest: ReviewManifest = {
  schemaVersion: 1,
  fingerprint: 'fingerprint',
  posts: [],
  skipped: [],
  counts: {
    selectedPosts: 0,
    selectedImages: 0,
    transformedPosts: 0,
    warningPosts: 0,
    skippedPosts: 0,
  },
}

describe('worker operation protocol (UT-018)', () => {
  it('fences stale results after a newer inspect operation', async () => {
    const worker = new FakeWorker()
    const client = new ArchiveWorkerClient(worker, () => 'operation-1')
    const first = client.inspect(new File(['a'], 'a.zip'))
    client.setOperationIdFactory(() => 'operation-2')
    const second = client.inspect(new File(['b'], 'b.zip'))

    worker.emit({
      type: 'manifest',
      operationId: 'operation-1',
      manifest,
    })
    worker.emit({
      type: 'manifest',
      operationId: 'operation-2',
      manifest,
    })

    await expect(second).resolves.toEqual(manifest)
    await expect(first).rejects.toThrow('operationSuperseded')
  })

  it('sends cancellation for the active operation', async () => {
    const worker = new FakeWorker()
    const client = new ArchiveWorkerClient(worker, () => 'operation-1')
    const pending = client.inspect(new File(['a'], 'a.zip'))
    client.cancel()
    expect(worker.sent.at(-1)).toEqual({
      type: 'cancel',
      operationId: 'operation-1',
    })
    await expect(pending).rejects.toThrow('operationCanceled')
  })

  it('cancels one sanitize request without discarding the active archive', async () => {
    const worker = new FakeWorker()
    const ids = vi
      .fn<() => string>()
      .mockReturnValueOnce('operation-1')
      .mockReturnValueOnce('sanitize-1')
      .mockReturnValueOnce('sanitize-2')
    const client = new ArchiveWorkerClient(worker, ids)
    const inspection = client.inspect(new File(['a'], 'a.zip'))
    worker.emit({
      type: 'manifest',
      operationId: 'operation-1',
      manifest,
    })
    await inspection

    const controller = new AbortController()
    const first = client.sanitize('media-1', controller.signal)
    controller.abort()

    await expect(first).rejects.toMatchObject({ name: 'AbortError' })
    expect(worker.sent.at(-1)).toEqual({
      type: 'cancelSanitize',
      operationId: 'operation-1',
      requestId: 'sanitize-1',
    })

    const second = client.sanitize('media-2')
    expect(worker.sent.at(-1)).toEqual({
      type: 'sanitize',
      operationId: 'operation-1',
      requestId: 'sanitize-2',
      token: 'media-2',
    })
    worker.emit({
      type: 'sanitized',
      operationId: 'operation-1',
      requestId: 'sanitize-2',
      buffer: new ArrayBuffer(1),
      mime: 'image/png',
      width: 1,
      height: 1,
    })
    await expect(second).resolves.toMatchObject({
      mime: 'image/png',
      width: 1,
      height: 1,
    })
  })
})
