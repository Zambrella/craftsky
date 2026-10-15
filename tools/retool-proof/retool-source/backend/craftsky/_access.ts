/**
 * Private draft gate for the CraftSky moderation integration proof.
 *
 * Operations are allowed only for the authenticated stable user who owns this
 * private draft. The id is derived from req.user (trusted session identity),
 * never from frontend parameters.
 *
 * Allowed: Doug Todd (doug@craftsky.social) — stable Retool user id 2691189.
 */
const ALLOWED_USER_IDS: readonly number[] = [2691189]

export function requireDraftOwner(user: User): void {
  if (!ALLOWED_USER_IDS.includes(user.id)) {
    throw new Error(
      'Forbidden: this stage 1 proof is restricted to the private draft owner (authenticated stable user id mismatch).',
    )
  }
}
