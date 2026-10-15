import { requireDraftOwner } from './_access'

/**
 * Stage 1 queue read: GET /v1/admin/moderation/cases on the synthetic
 * fixture. No state filter is sent so resolved cases remain reachable after
 * an apply/reversal round trip. Returns the typed response body unchanged.
 */
export default async function (req: { user: User }) {
  requireDraftOwner(req.user)
  const result = await craftskySyntheticModerationProof.moderation.listModerationCases({
    limit: 20,
  })
  return result.data
}
