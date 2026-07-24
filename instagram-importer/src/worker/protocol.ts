import type { ReviewManifest } from '../domain/types'
import type { SupportedImageMime } from '../media/inspect'
import type { SafeErrorCode } from '../privacy/safeDiagnostics'

export type WorkerRequest =
  | {
      readonly type: 'inspect'
      readonly operationId: string
      readonly file: File
      readonly now: number
    }
  | {
      readonly type: 'sanitize'
      readonly operationId: string
      readonly requestId: string
      readonly token: string
    }
  | {
      readonly type: 'cancelSanitize'
      readonly operationId: string
      readonly requestId: string
    }
  | {
      readonly type: 'cancel'
      readonly operationId: string
    }

export type WorkerResponse =
  | {
      readonly type: 'progress'
      readonly operationId: string
      readonly phase:
        | 'directory'
        | 'metadata'
        | 'mediaPreflight'
        | 'sanitizing'
      readonly completed: number
      readonly total?: number
    }
  | {
      readonly type: 'manifest'
      readonly operationId: string
      readonly manifest: ReviewManifest
    }
  | {
      readonly type: 'sanitized'
      readonly operationId: string
      readonly requestId: string
      readonly buffer: ArrayBuffer
      readonly mime: SupportedImageMime
      readonly width: number
      readonly height: number
    }
  | {
      readonly type: 'error'
      readonly operationId: string
      readonly requestId?: string
      readonly code: SafeErrorCode
    }
