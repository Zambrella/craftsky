/// <reference lib="webworker" />

import type { ZipReader } from '@zip.js/zip.js'

import { inspectArchive, sanitizeArchiveEntry } from '../archive/inspectArchive'
import { safeWorkerErrorCode } from './errorCode'
import type { WorkerRequest, WorkerResponse } from './protocol'

declare const self: DedicatedWorkerGlobalScope

let active:
  | {
      readonly operationId: string
      readonly controller: AbortController
      readonly reader: ZipReader<Blob>
      readonly tokenEntries: Awaited<
        ReturnType<typeof inspectArchive>
      >['tokenEntries']
    }
  | undefined
let inspecting:
  | {
      readonly operationId: string
      readonly controller: AbortController
    }
  | undefined
let sanitizeQueue: Promise<void> = Promise.resolve()
const sanitizing = new Map<
  string,
  {
    readonly operationId: string
    readonly controller: AbortController
  }
>()

function cancelSanitizing(operationId: string): void {
  for (const [requestId, operation] of sanitizing) {
    if (operation.operationId !== operationId) continue
    operation.controller.abort()
    sanitizing.delete(requestId)
  }
}

function respond(message: WorkerResponse, transfer: Transferable[] = []): void {
  self.postMessage(message, transfer)
}

self.addEventListener('message', (event: MessageEvent<WorkerRequest>) => {
  const request = event.data
  if (request.type === 'cancel') {
    cancelSanitizing(request.operationId)
    if (inspecting?.operationId === request.operationId) {
      inspecting.controller.abort()
      inspecting = undefined
    }
    if (active?.operationId === request.operationId) {
      active.controller.abort()
      void active.reader.close()
      active = undefined
    }
    return
  }
  if (request.type === 'cancelSanitize') {
    const operation = sanitizing.get(request.requestId)
    if (operation?.operationId === request.operationId) {
      operation.controller.abort()
      sanitizing.delete(request.requestId)
    }
    return
  }
  if (request.type === 'inspect') {
    void (async () => {
      if (inspecting) {
        inspecting.controller.abort()
        inspecting = undefined
      }
      if (active) {
        cancelSanitizing(active.operationId)
        active.controller.abort()
        await active.reader.close().catch(() => undefined)
        active = undefined
      }
      const controller = new AbortController()
      inspecting = {
        operationId: request.operationId,
        controller,
      }
      try {
        const inspection = await inspectArchive(
          request.file,
          new Date(request.now),
          controller.signal,
          (phase, completed, total) => {
            respond({
              type: 'progress',
              operationId: request.operationId,
              phase,
              completed,
              total,
            })
          },
        )
        if (controller.signal.aborted) {
          await inspection.reader.close().catch(() => undefined)
          return
        }
        if (inspecting?.operationId !== request.operationId) {
          await inspection.reader.close().catch(() => undefined)
          return
        }
        inspecting = undefined
        active = {
          operationId: request.operationId,
          controller,
          reader: inspection.reader,
          tokenEntries: inspection.tokenEntries,
        }
        respond({
          type: 'manifest',
          operationId: request.operationId,
          manifest: inspection.manifest,
        })
      } catch (error) {
        if (inspecting?.operationId === request.operationId) {
          inspecting = undefined
        }
        if (controller.signal.aborted) return
        respond({
          type: 'error',
          operationId: request.operationId,
          code: safeWorkerErrorCode(error),
        })
      }
    })()
    return
  }
  const controller = new AbortController()
  sanitizing.set(request.requestId, {
    operationId: request.operationId,
    controller,
  })
  sanitizeQueue = sanitizeQueue.then(async () => {
    if (
      controller.signal.aborted ||
      !active ||
      active.operationId !== request.operationId
    ) {
      sanitizing.delete(request.requestId)
      return
    }
    const entry = active.tokenEntries.get(request.token)
    if (!entry) {
      sanitizing.delete(request.requestId)
      respond({
        type: 'error',
        operationId: request.operationId,
        requestId: request.requestId,
        code: 'mediaUnavailable',
      })
      return
    }
    try {
      respond({
        type: 'progress',
        operationId: request.operationId,
        phase: 'sanitizing',
        completed: 0,
        total: 1,
      })
      const sanitized = await sanitizeArchiveEntry(
        entry,
        controller.signal,
      )
      if (
        controller.signal.aborted ||
        !active ||
        active.operationId !== request.operationId
      ) {
        return
      }
      respond(
        {
          type: 'sanitized',
          operationId: request.operationId,
          requestId: request.requestId,
          ...sanitized,
        },
        [sanitized.buffer],
      )
    } catch {
      if (controller.signal.aborted) return
      respond({
        type: 'error',
        operationId: request.operationId,
        requestId: request.requestId,
        code: 'mediaUnavailable',
      })
    } finally {
      sanitizing.delete(request.requestId)
    }
  })
})
