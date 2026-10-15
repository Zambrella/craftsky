import type { CaseSummary } from '../../utils/craftskyTypes'
import { Inbox } from 'lucide-react'

type Props = {
  cases: CaseSummary[]
  loading: boolean
  error: string | null
  selectedCaseReference: string | null
  onSelect: (caseReference: string) => void
  /** True while a command is dispatching or post-success reads are settling. */
  busy: boolean
}

/** Case queue list. Data comes only from the synthetic fixture. */
export function CaseQueueSection({ cases, loading, error, selectedCaseReference, onSelect, busy }: Props) {
  return (
    <section
      aria-labelledby="queue-heading"
      className="rounded-lg border border-border bg-card shadow-retool-sm"
    >
      <div className="border-b border-border px-5 py-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 id="queue-heading" className="text-base font-semibold">
            Cases
          </h2>
          <span className="text-xs uppercase tracking-wide text-muted-foreground">
            Synthetic development data
          </span>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Cases from the isolated synthetic CraftSky fixture.
        </p>
      </div>
      <div className="p-4">
        {error !== null ? (
          <p role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            Queue read failed: {error}
          </p>
        ) : loading && cases.length === 0 ? (
          <div className="flex items-center gap-2 px-2 py-6 text-sm text-muted-foreground">
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
            Loading queue…
          </div>
        ) : cases.length === 0 ? (
          <div className="flex items-center gap-3 px-2 py-6 text-sm text-muted-foreground">
            <Inbox className="h-5 w-5 shrink-0" aria-hidden="true" />
            No cases reported by the fixture.
          </div>
        ) : (
          <ul className="space-y-2" aria-label="Cases">
            {cases.map((c) => {
              const selected = c.caseReference === selectedCaseReference
              return (
                <li key={c.caseReference}>
                  <button
                    type="button"
                    aria-pressed={selected}
                    onClick={() => onSelect(c.caseReference)}
                    disabled={busy}
                    className={`w-full rounded-md border px-4 py-3 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:cursor-not-allowed disabled:opacity-60 ${
                      selected
                        ? 'border-primary bg-accent'
                        : 'border-border hover:bg-accent/60'
                    }`}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="font-mono text-sm">{c.caseReference}</span>
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          c.state === 'open'
                            ? 'bg-accent text-accent-foreground'
                            : 'bg-secondary text-secondary-foreground'
                        }`}
                      >
                        {c.state}
                      </span>
                    </div>
                    <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <span>revision {c.revision}</span>
                      <span>{c.subjectType ?? 'unknown subject type'}</span>
                      <span>{c.reportCount ?? 0} report(s)</span>
                    </div>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </section>
  )
}
