import { useCallback, useEffect, useRef, useState } from 'react'
import {
  useApplyDecision,
  useCheckEvidenceTransport,
  useGetCase,
  useGetIdentity,
  useListCases,
  useReverseEffects,
} from '../hooks/backend/craftsky'
import { useModerationActions } from '../hooks/useModerationActions'
import { ActionPanel } from './ui/ActionPanel'
import { CaseDetailSection } from './ui/CaseDetailSection'
import { CaseQueueSection } from './ui/CaseQueueSection'
import { EvidenceCheck } from './ui/EvidenceCheck'
import { OperatorIdentitySection } from './ui/OperatorIdentitySection'
import {
  asCaseDetail,
  asCaseList,
  asIdentity,
  asMutationResult,
  asTransportResult,
  type CaseDetail,
  type EvidenceTransportResult,
  type IdentityResult,
} from '../utils/craftskyTypes'

/**
 * Stage 1: CraftSky moderation integration proof against the isolated
 * synthetic fixture. Private draft; nothing is published. Reads and the two
 * confirmed synthetic actions only — no other resources, no AI at runtime.
 */
export default function ModerationProof() {
  const identityHook = useGetIdentity({ cachePolicy: 'stale-while-revalidate' })
  const casesHook = useListCases({ cachePolicy: 'stale-while-revalidate' })
  const caseHook = useGetCase({ cachePolicy: 'stale-while-revalidate' })
  const transportHook = useCheckEvidenceTransport()
  const applyHook = useApplyDecision()
  const reverseHook = useReverseEffects()

  const [selectedCaseReference, setSelectedCaseReference] = useState<string | null>(null)
  const selectedRef = useRef<string | null>(null)
  selectedRef.current = selectedCaseReference

  const identity: IdentityResult | null = asIdentity(identityHook.data)
  const caseList = asCaseList(casesHook.data)
  const detail: CaseDetail | null = asCaseDetail(caseHook.data)
  const transport: EvidenceTransportResult | null =
    transportHook.data === undefined || transportHook.data === null
      ? null
      : asTransportResult(transportHook.data)

  // Initial reads: identity discovers the fixture's current case/evidence ids.
  useEffect(() => {
    void identityHook.trigger().result
    void casesHook.trigger().result
    // Hooks are stable across renders; run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Default selection: the identity-reported fixture case, else the first listed.
  useEffect(() => {
    if (selectedCaseReference !== null) return
    const candidate = identity?.caseReference ?? caseList?.items[0]?.caseReference
    if (candidate !== undefined) setSelectedCaseReference(candidate)
  }, [identity, caseList, selectedCaseReference])

  // Load detail whenever the selection changes.
  useEffect(() => {
    if (selectedCaseReference === null) return
    void caseHook.trigger({ caseReference: selectedCaseReference }).result
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCaseReference])

  const refreshSelection = useCallback(async () => {
    const ref = selectedRef.current
    // allSettled: busy is released only once BOTH reads have settled, and any
    // rejection stays owned by the sections' error channels — it can never
    // relabel a committed command as failed.
    await Promise.allSettled([
      casesHook.trigger(undefined, { skipCache: true }).result,
      ref !== null ? caseHook.trigger({ caseReference: ref }, { skipCache: true }).result : Promise.resolve(null),
    ])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const runApply = useCallback(
    async (p: { idempotencyKey: string; expectedRevision: number }) =>
      asMutationResult(await applyHook.trigger({ ...p }).result),
    [applyHook.trigger],
  )
  const runReverse = useCallback(
    async (p: { idempotencyKey: string; expectedRevision: number }) =>
      asMutationResult(await reverseHook.trigger({ ...p }).result),
    [reverseHook.trigger],
  )

  const actions = useModerationActions({ runApply, runReverse, onSettledSuccess: refreshSelection })
  // While a command is dispatching or post-success reads are settling, block
  // conflicting controls (queue selection, detail refresh) as well as actions.
  const readsBusy = actions.dispatching || actions.settling

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="mx-auto max-w-6xl px-6 py-8">
        <header className="mb-6">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-xl font-semibold tracking-tight">
              CraftSky moderation integration proof
            </h1>
            <span className="rounded-full border border-border bg-secondary px-3 py-1 text-xs font-medium uppercase tracking-wide text-secondary-foreground">
              Synthetic development data
            </span>
          </div>
          <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
            Stage 1 integration proof on the isolated synthetic CraftSky fixture.
            The API operator is a fixed synthetic alpha credential (proof-alpha);
            the Retool current user is shown separately in the identity panel.
            Per-user custom auth remains unverified. Private draft — not
            published; no other resources and no AI features are used at runtime.
          </p>
        </header>

        {identityHook.error !== null ? (
          <p role="alert" className="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            Identity read failed: {identityHook.error}
          </p>
        ) : null}

        {identity === null && identityHook.loading ? (
          <p className="mb-4 flex items-center gap-2 text-sm text-muted-foreground">
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
            Reading identity from the synthetic fixture…
          </p>
        ) : null}

        <div className="grid gap-6 lg:grid-cols-[3fr_2fr]">
          <div className="flex flex-col gap-6">
            <CaseQueueSection
              cases={caseList?.items ?? []}
              loading={casesHook.loading}
              error={casesHook.error}
              selectedCaseReference={selectedCaseReference}
              onSelect={setSelectedCaseReference}
              busy={readsBusy}
            />
            <CaseDetailSection
              detail={detail}
              loading={caseHook.loading}
              error={caseHook.error}
              onRefresh={() => {
                if (selectedCaseReference !== null) {
                  void caseHook
                    .trigger({ caseReference: selectedCaseReference }, { skipCache: true })
                    .result
                }
              }}
              refreshing={caseHook.loading}
              busy={readsBusy}
            />
            <ActionPanel
              detail={detail}
              applyState={actions.states.apply}
              reverseState={actions.states.reverse}
              anyRunning={actions.anyRunning}
              onBegin={actions.begin}
              onCancel={actions.cancel}
              onConfirm={(kind) => {
                if (detail !== null) void actions.execute(kind, detail.revision)
              }}
              onDismiss={actions.dismiss}
              onConflictRefresh={(kind) => {
                actions.dismiss(kind)
                void refreshSelection()
              }}
            />
          </div>
          <div className="flex flex-col gap-6">
            {identity !== null ? <OperatorIdentitySection identity={identity} /> : null}
            <section aria-labelledby="notes-heading" className="rounded-lg border border-border bg-card p-5 shadow-retool-sm">
              <h2 id="notes-heading" className="text-base font-semibold">
                Scope notes
              </h2>
              <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                <li>Only the CraftSky synthetic moderation proof resource is used.</li>
                <li>Queue, detail and evidence data are synthetic fixtures; labels mark them as such.</li>
                <li>Actions require explicit confirmation; failures surface code and requestId verbatim.</li>
              </ul>
              <div className="mt-4">
                <EvidenceCheck
                  onRun={() => void transportHook.trigger(undefined, { skipCache: true }).result}
                  running={transportHook.loading}
                  result={transport}
                  error={transportHook.error}
                />
              </div>
            </section>
          </div>
        </div>
      </div>
    </div>
  )
}
