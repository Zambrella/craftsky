import type { ReviewManifest } from '../domain/types'
import type { SupportedImageMime } from '../media/inspect'
import type { WorkerRequest, WorkerResponse } from './protocol'

export interface WorkerPort {
  postMessage(message: WorkerRequest, transfer?: Transferable[]): void
  addEventListener(
    type: 'message',
    listener: (event: MessageEvent<WorkerResponse>) => void,
  ): void
  removeEventListener(
    type: 'message',
    listener: (event: MessageEvent<WorkerResponse>) => void,
  ): void
  terminate?(): void
}

export interface SanitizedMedia {
  readonly buffer: ArrayBuffer
  readonly mime: SupportedImageMime
  readonly width: number
  readonly height: number
}

type OperationIdFactory = () => string

function defaultId(): string {
  return crypto.randomUUID()
}

export class ArchiveWorkerClient {
  private activeOperationId?: string
  private operationIdFactory: OperationIdFactory
  private inspectResolve?: (manifest: ReviewManifest) => void
  private inspectReject?: (error: Error) => void
  private readonly sanitizeRequests = new Map<
    string,
    {
      resolve: (media: SanitizedMedia) => void
      reject: (error: Error) => void
      signal?: AbortSignal
      abortListener?: () => void
    }
  >()
  private progressListener?: (response: Extract<WorkerResponse, { type: 'progress' }>) => void

  constructor(
    private readonly worker: WorkerPort,
    operationIdFactory: OperationIdFactory = defaultId,
  ) {
    this.operationIdFactory = operationIdFactory
    worker.addEventListener('message', this.onMessage)
  }

  setOperationIdFactory(factory: OperationIdFactory): void {
    this.operationIdFactory = factory
  }

  inspect(
    file: File,
    onProgress?: (
      response: Extract<WorkerResponse, { type: 'progress' }>,
    ) => void,
  ): Promise<ReviewManifest> {
    if (this.activeOperationId) {
      this.worker.postMessage({
        type: 'cancel',
        operationId: this.activeOperationId,
      })
      this.inspectReject?.(new Error('operationSuperseded'))
      this.rejectSanitizeRequests('operationSuperseded')
    }
    const operationId = this.operationIdFactory()
    this.activeOperationId = operationId
    this.progressListener = onProgress
    const promise = new Promise<ReviewManifest>((resolve, reject) => {
      this.inspectResolve = resolve
      this.inspectReject = reject
    })
    this.worker.postMessage({
      type: 'inspect',
      operationId,
      file,
      now: Date.now(),
    })
    return promise
  }

  sanitize(
    token: string,
    signal?: AbortSignal,
  ): Promise<SanitizedMedia> {
    const operationId = this.activeOperationId
    if (!operationId) return Promise.reject(new Error('noActiveArchive'))
    const requestId = this.operationIdFactory()
    const promise = new Promise<SanitizedMedia>((resolve, reject) => {
      const abortListener = signal
        ? () => {
            const pending = this.takeSanitizeRequest(requestId)
            if (!pending) return
            this.worker.postMessage({
              type: 'cancelSanitize',
              operationId,
              requestId,
            })
            pending.reject(
              new DOMException('The operation was aborted.', 'AbortError'),
            )
          }
        : undefined
      this.sanitizeRequests.set(requestId, {
        resolve,
        reject,
        signal,
        abortListener,
      })
      signal?.addEventListener('abort', abortListener!, { once: true })
    })
    if (signal?.aborted) {
      const pending = this.takeSanitizeRequest(requestId)
      pending?.reject(
        new DOMException('The operation was aborted.', 'AbortError'),
      )
      return promise
    }
    this.worker.postMessage({
      type: 'sanitize',
      operationId,
      requestId,
      token,
    })
    return promise
  }

  cancel(): void {
    if (!this.activeOperationId) return
    this.worker.postMessage({
      type: 'cancel',
      operationId: this.activeOperationId,
    })
    this.inspectReject?.(new Error('operationCanceled'))
    this.rejectSanitizeRequests('operationCanceled')
    this.clearActive()
  }

  dispose(): void {
    this.cancel()
    this.worker.removeEventListener('message', this.onMessage)
    this.worker.terminate?.()
  }

  private readonly onMessage = (
    event: MessageEvent<WorkerResponse>,
  ): void => {
    const response = event.data
    if (response.operationId !== this.activeOperationId) return
    if (response.type === 'progress') {
      this.progressListener?.(response)
      return
    }
    if (response.type === 'manifest') {
      this.inspectResolve?.(response.manifest)
      this.inspectResolve = undefined
      this.inspectReject = undefined
      return
    }
    if (response.type === 'sanitized') {
      const pending = this.takeSanitizeRequest(response.requestId)
      if (!pending) return
      pending.resolve({
        buffer: response.buffer,
        mime: response.mime,
        width: response.width,
        height: response.height,
      })
      return
    }
    const error = new Error(response.code)
    if (response.requestId) {
      const pending = this.takeSanitizeRequest(response.requestId)
      pending?.reject(error)
    } else {
      this.inspectReject?.(error)
      this.clearActive()
    }
  }

  private rejectSanitizeRequests(code: string): void {
    for (const request of this.sanitizeRequests.values()) {
      request.signal?.removeEventListener(
        'abort',
        request.abortListener!,
      )
      request.reject(new Error(code))
    }
    this.sanitizeRequests.clear()
  }

  private takeSanitizeRequest(
    requestId: string,
  ):
    | {
        resolve: (media: SanitizedMedia) => void
        reject: (error: Error) => void
        signal?: AbortSignal
        abortListener?: () => void
      }
    | undefined {
    const request = this.sanitizeRequests.get(requestId)
    if (!request) return undefined
    this.sanitizeRequests.delete(requestId)
    request.signal?.removeEventListener(
      'abort',
      request.abortListener!,
    )
    return request
  }

  private clearActive(): void {
    this.activeOperationId = undefined
    this.inspectResolve = undefined
    this.inspectReject = undefined
    this.progressListener = undefined
  }
}

export function createArchiveWorkerClient(): ArchiveWorkerClient {
  const worker = new Worker(new URL('./archive.worker.ts', import.meta.url), {
    type: 'module',
    name: 'instagram-archive',
  })
  return new ArchiveWorkerClient(worker)
}
