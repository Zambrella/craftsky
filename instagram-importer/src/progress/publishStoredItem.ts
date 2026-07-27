import type { ReviewPost } from '../domain/types'
import {
  publishPost,
  type PublishOutcome,
  type RepoClient,
  type SanitizedMedia,
} from '../pds/publisher'
import { withTransientRetry } from '../pds/retry'
import type { ImportItemRow } from './state'

export async function publishStoredItem(input: {
  readonly stored: ImportItemRow
  readonly repo: RepoClient
  readonly did: string
  readonly post: ReviewPost
  readonly sanitize: (
    token: string,
    signal?: AbortSignal,
  ) => Promise<SanitizedMedia>
  readonly signal?: AbortSignal
}): Promise<PublishOutcome> {
  return withTransientRetry(
    () =>
      publishPost({
        repo: input.repo,
        did: input.did,
        post: input.post,
        sanitize: (token) => input.sanitize(token, input.signal),
        ownedCreatedCid: input.stored.createdCid,
      }),
    { signal: input.signal },
  )
}
