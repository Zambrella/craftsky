import type { EvidenceTransportResult } from '../../utils/craftskyTypes'
import { ShieldCheck } from 'lucide-react'

type Props = {
  onRun: () => void
  running: boolean
  result: EvidenceTransportResult | null
  error: string | null
}

/**
 * Small benign transport probe: POSTs the fixed benign reason to the
 * fixture's evidence access endpoint (one-pixel PNG, no restricted data) and
 * shows structural metadata only — never PNG content, never header metadata.
 */
export function EvidenceCheck({ onRun, running, result, error }: Props) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Benign evidence transport check
        </h3>
        <button
          type="button"
          onClick={onRun}
          disabled={running}
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
        >
          <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
          {running ? 'Checking…' : 'Run transport check'}
        </button>
      </div>
      {error !== null ? (
        <p role="alert" className="mt-2 text-xs text-destructive">Probe failed: {error}</p>
      ) : null}
      {result !== null ? (
        result.ok ? (
          <p className="mt-2 text-xs text-muted-foreground">
            Binary envelope verified — type <span className="font-mono">{result.contentType}</span>,{' '}
            decoded {result.decodedByteLength} bytes (declared {result.declaredSizeBytes}, match:{' '}
            {result.sizeMatches ? 'yes' : 'no'}). Buffered JSON envelope with base64; StreamRaw emits
            a JSON metadata envelope, so direct binary passthrough is not suitable. Content is never
            displayed or stored here.
          </p>
        ) : (
          <p role="alert" className="mt-2 text-xs text-destructive">
            Probe rejected: {result.message}
            {result.code ? ` · code: ${result.code}` : ''}
            {result.requestId ? ` · requestId: ${result.requestId}` : ''}
          </p>
        )
      ) : null}
    </div>
  )
}
