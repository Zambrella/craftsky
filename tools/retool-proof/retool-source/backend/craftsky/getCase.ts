import { requireDraftOwner } from './_access'

type Params = {
  caseReference: string
}

/**
 * Stage 1 case detail read: GET /v1/admin/moderation/cases/{caseReference}
 * on the synthetic fixture. Returns the typed response body unchanged so the
 * UI can display the authoritative state and revision.
 */
export default async function (req: { params: Params; user: User }) {
  requireDraftOwner(req.user)
  const result = await craftskySyntheticModerationProof.moderation.getModerationCase(
    req.params.caseReference,
  )
  return result.data
}
