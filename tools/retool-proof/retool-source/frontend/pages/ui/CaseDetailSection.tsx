import type { CaseDetail } from '../../utils/craftskyTypes'
import { FileText } from 'lucide-react'

type Props = {
  detail: CaseDetail | null
  loading: boolean
  error: string | null
  onRefresh: () => void
  refreshing: boolean
  /** True while a command is dispatching or post-success reads are settling. */
  busy: boolean
}

/** Case detail pane; the displayed revision is what commands will use. */
export function CaseDetailSection({ detail, loading, error, onRefresh, refreshing, busy }: Props) {
  if (error !== null) {
    return (
      <section aria-labelledby="detail-heading" className="rounded-lg border border-border bg-card p-5 shadow-retool-sm">
        <h2 id="detail-heading" className="text-base font-semibold">Case detail</h2>
        <p role="alert" className="mt-3 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          Detail read failed: {error}
        </p>
        <button
          type="button"
          onClick={onRefresh}
          disabled={refreshing || busy}
          className="mt-3 rounded-md border border-border px-3 py-1.5 text-sm font-medium hover:bg-accent disabled:opacity-50"
        >
          Retry read
        </button>
      </section>
    )
  }

  if (detail === null) {
    return (
      <section aria-labelledby="detail-heading" className="rounded-lg border border-border bg-card p-5 shadow-retool-sm">
        <h2 id="detail-heading" className="text-base font-semibold">Case detail</h2>
        <div className="mt-3 flex items-center gap-2 text-sm text-muted-foreground">
          {loading ? (
            <>
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
              Loading case…
            </>
          ) : (
            <>Select a case from the queue.</>
          )}
        </div>
      </section>
    )
  }

  return (
    <section
      aria-labelledby="detail-heading"
      className="rounded-lg border border-border bg-card shadow-retool-sm"
    >
      <div className="border-b border-border px-5 py-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 id="detail-heading" className="flex items-center gap-2 text-base font-semibold">
            <FileText className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            Case detail
          </h2>
          <button
            type="button"
            onClick={onRefresh}
            disabled={refreshing || busy}
            className="rounded-md border border-border px-3 py-1 text-sm font-medium hover:bg-accent disabled:opacity-50"
          >
            {refreshing ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>
        <p className="mt-1 font-mono text-sm">{detail.caseReference}</p>
      </div>

      <dl className="grid gap-x-6 gap-y-2 px-5 py-4 text-sm sm:grid-cols-2">
        <div className="flex justify-between gap-4 sm:block">
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">State</dt>
          <dd className="font-medium">{detail.state}</dd>
        </div>
        <div className="flex justify-between gap-4 sm:block">
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">
            Expected revision for commands
          </dt>
          <dd className="font-mono text-base font-semibold text-primary">{detail.revision}</dd>
        </div>
        <div className="flex justify-between gap-4 sm:block">
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">Subject</dt>
          <dd className="font-mono text-xs">{detail.subjectDid ?? '—'}</dd>
        </div>
        <div className="flex justify-between gap-4 sm:block">
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">Standing</dt>
          <dd>
            {detail.standing
              ? `${detail.standing.activeStrikeCount ?? 0} active strike(s) / threshold ${detail.standing.strikeThreshold ?? '—'}`
              : '—'}
          </dd>
        </div>
      </dl>

      <div className="space-y-4 border-t border-border px-5 py-4">
        <DetailList title="Reports" empty="No reports recorded.">
          {detail.reports?.map((r, i) => (
            <li key={`${r.id ?? i}`} className="text-sm">
              <span className="font-medium">{r.reason ?? 'unspecified reason'}</span>
              <span className="text-muted-foreground"> — {r.details ?? 'no details'} (by {r.reporterDid ?? 'unknown'})</span>
            </li>
          ))}
        </DetailList>

        <DetailList title="Decisions" empty="No decisions recorded.">
          {detail.decisions?.map((d, i) => (
            <li key={i} className="text-sm">
              <span className="font-medium">{d.disposition ?? 'unknown'}</span>
              <span className="text-muted-foreground"> — {d.reason ?? 'no reason'} ({d.evidenceNotes ?? 'no notes'})</span>
            </li>
          ))}
        </DetailList>

        <DetailList title="Effects" empty="No effects recorded.">
          {detail.effects?.map((e, i) => (
            <li key={`${e.effectId ?? i}-${e.action ?? i}`} className="text-sm">
              <span className="font-mono text-xs">{e.type ?? 'unknown type'}</span>{' '}
              <span className="text-muted-foreground">
                {e.action ?? 'unknown action'}, active: {e.active === true ? 'yes' : 'no'}
                {e.rationale ? `, rationale: ${e.rationale}` : ''}
              </span>
            </li>
          ))}
        </DetailList>

        <DetailList title="Events" empty="No events recorded.">
          {detail.events?.map((ev, i) => (
            <li key={i} className="text-sm">
              <span className="font-medium">{ev.type ?? 'unknown'}</span>
              <span className="text-muted-foreground">
                {' '}rev {ev.revision ?? '—'} by {ev.actorId ?? 'unknown'} ({ev.sourceSystem ?? 'unknown source'})
              </span>
            </li>
          ))}
        </DetailList>
      </div>
    </section>
  )
}

function DetailList({
  title,
  empty,
  children,
}: {
  title: string
  empty: string
  children: React.ReactNode
}) {
  const hasItems = Array.isArray(children) ? children.length > 0 : children !== null
  return (
    <div>
      <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{title}</h3>
      {hasItems ? (
        <ul className="mt-2 space-y-1.5">{children}</ul>
      ) : (
        <p className="mt-1 text-sm text-muted-foreground">{empty}</p>
      )}
    </div>
  )
}
