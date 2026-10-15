import { requireDraftOwner } from './_access'

type IdentityResult = {
  /** Retool current user, derived server-side from the trusted session. */
  retoolUser: {
    stableId: number
    fullName: string
    email: string
  }
  /** Fixed synthetic proof operator credential — NOT a per-user identity. */
  apiOperator: {
    actorId: string
    role?: string
    synthetic: boolean
    permissions?: Record<string, boolean>
  }
  /** Current synthetic fixture references returned by the identity endpoint. */
  caseReference: string
  evidenceId: string
}

/**
 * Stage 1 identity read: GET /v1/admin/proof/identity on the synthetic
 * fixture. Returns the Retool stable user id (from req.user) and the API
 * operator actorId under distinct labels. proof-alpha is a fixed proof
 * credential; this pairing does not prove shared production per-user custom
 * auth.
 */
export default async function (req: { user: User }): Promise<IdentityResult> {
  requireDraftOwner(req.user)
  const result = await craftskySyntheticModerationProof.proof.getProofIdentity()
  return {
    retoolUser: {
      stableId: req.user.id,
      fullName: req.user.fullName,
      email: req.user.email,
    },
    apiOperator: {
      actorId: result.data.actorId,
      role: result.data.role,
      synthetic: result.data.synthetic,
      permissions: result.data.permissions,
    },
    caseReference: result.data.caseReference,
    evidenceId: result.data.evidenceId,
  }
}
