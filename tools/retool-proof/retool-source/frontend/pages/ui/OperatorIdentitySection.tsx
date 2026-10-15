import type { IdentityResult } from '../../utils/craftskyTypes'
import { UserRound } from 'lucide-react'

/**
 * Operator identity panel. Shows the trusted Retool session user and the
 * synthetic fixture's API operator under explicit, distinct labels.
 */
export function OperatorIdentitySection({ identity }: { identity: IdentityResult }) {
  return (
    <section
      aria-labelledby="operator-heading"
      className="rounded-lg border border-border bg-card p-5 shadow-retool-sm"
    >
      <div className="flex items-center gap-3">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-secondary text-secondary-foreground">
          <UserRound className="h-5 w-5" aria-hidden="true" />
        </span>
        <h2 id="operator-heading" className="text-base font-semibold">
          Operator identity
        </h2>
      </div>

      <div className="mt-4 space-y-4">
        <div className="rounded-md border border-border p-3">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Retool current user (trusted session, server-derived)
          </h3>
          <dl className="mt-2 space-y-1 text-sm">
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">Stable user id</dt>
              <dd className="font-mono">{identity.retoolUser.stableId}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">Name</dt>
              <dd>{identity.retoolUser.fullName}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">Email</dt>
              <dd>{identity.retoolUser.email}</dd>
            </div>
          </dl>
        </div>

        <div className="rounded-md border border-border p-3">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            API operator (synthetic fixture)
          </h3>
          <dl className="mt-2 space-y-1 text-sm">
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">actorId</dt>
              <dd className="font-mono">{identity.apiOperator.actorId}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">Role</dt>
              <dd>{identity.apiOperator.role ?? '—'}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">Synthetic</dt>
              <dd>{identity.apiOperator.synthetic ? 'yes' : 'no'}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted-foreground">Permissions</dt>
              <dd className="font-mono text-xs">
                {Object.keys(identity.apiOperator.permissions ?? {}).join(', ') || '—'}
              </dd>
            </div>
          </dl>
        </div>
      </div>

      <p className="mt-4 text-xs text-muted-foreground">
        proof-alpha is a fixed proof credential scoped to the synthetic fixture.
        This pairing demonstrates the integration path only and does not prove
        shared production per-user custom auth.
      </p>
    </section>
  )
}
