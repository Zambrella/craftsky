import { expect, test, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CasesWorkspace } from '../src/cases/CasesWorkspace'

const reference = 'MOD-50000000-0000-4000-8000-000000000001'
const operator = { did: 'did:plc:operator', actorId: 'operator:fixture', permissionSet: 'admin' as const, permissions: { 'moderation.read': true, 'moderation.manage': true }, assignedIncidentReferences: [] }
const initial = { caseReference: reference, state: 'open', origin: 'userReport', revision: 0, subjectType: 'post', subjectDid: 'did:plc:author', ownerDid: 'did:plc:author', reportCount: 1, createdAt: '2026-10-15T10:00:00Z', reports: [{ id: 'report-1', reporterDid: 'did:plc:reporter', reason: 'spam', details: 'Synthetic report for review', createdAt: '2026-10-15T10:00:00Z' }], events: [], decisions: [], effects: [], standing: { activeStrikeCount: 0, strikeThreshold: 3, thresholdSuspended: false, severeSuspended: false, suspended: false }, safeSnapshot: { type: 'post', did: 'did:plc:author' } }
function response(value: unknown, status = 200) { return new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } }) }
function fixture(override?: (path: string, init?: RequestInit) => Response | Promise<Response> | undefined) {
  return vi.stubGlobal('fetch', vi.fn(async (input: string, init?: RequestInit) => {
    const path = String(input)
    const custom = override?.(path, init); if (custom !== undefined) return custom
    if (path.endsWith('/auth/csrf')) return response({ csrfToken: 'fixture-csrf' })
    if (path.includes('/preview')) return response({ state: 'available', title: '', text: 'A synthetic craft project.', mediaAvailable: false })
    if (path.includes('/provenance')) return response({ items: [] })
    if (path.includes('/cases?')) return response({ items: [initial] })
    if (path.endsWith('/cases/' + reference)) return response(initial)
    throw new Error('Unexpected fixture endpoint')
  }))
}

test('operator filters the case queue and opens content, reports and standing', async () => {
  fixture()
  const user = userEvent.setup()
  render(<CasesWorkspace operator={operator} onSessionExpired={() => {}} />)
  await user.click(await screen.findByRole('link', { name: 'Review ' + reference }))
  expect(await screen.findByText('A synthetic craft project.')).toBeTruthy()
  expect(screen.getByText('Synthetic report for review')).toBeTruthy()
  expect(screen.getByRole('heading', { name: 'Account standing' })).toBeTruthy()
  await user.selectOptions(screen.getByLabelText('Case status'), 'resolved')
  await waitFor(() => expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).includes('state=resolved'))).toBe(true))
})

async function openCase() {
  const user = userEvent.setup()
  render(<CasesWorkspace operator={operator} onSessionExpired={() => {}} />)
  await user.click(await screen.findByRole('link', { name: 'Review ' + reference }))
  await screen.findByText('A synthetic craft project.')
  return user
}
async function prepareDecision(user: ReturnType<typeof userEvent.setup>) {
  await user.selectOptions(screen.getByLabelText('Decision'), 'violation')
  await user.selectOptions(screen.getByLabelText('Policy reason'), 'spam')
  await user.type(screen.getByLabelText('Internal evidence notes'), 'Reviewed the synthetic evidence.')
  await user.type(screen.getByLabelText('Explanation for the account owner'), 'Repeated unsolicited promotion.')
  await user.click(screen.getByLabelText('Hide content'))
  await user.click(screen.getByLabelText('Account strike'))
  await user.click(screen.getByRole('button', { name: 'Review decision' }))
}

test('consequence help works with hover and keyboard without selecting or submitting a decision', async () => {
  fixture()
  const user = await openCase()
  await user.selectOptions(screen.getByLabelText('Decision'), 'violation')
  expect(screen.getAllByRole('button', { name: /purpose and effect/ })).toHaveLength(6)
  const help = screen.getByRole('button', { name: 'Formal warning: purpose and effect' })
  const choice = screen.getByRole('checkbox', { name: 'Formal warning' }) as HTMLInputElement
  expect(screen.queryByRole('tooltip')).toBeNull()
  await user.hover(help)
  expect(screen.getByRole('tooltip').textContent).toContain('account owner')
  await user.click(help)
  expect(choice.checked).toBe(false)
  await user.unhover(help)
  expect(screen.getByRole('tooltip')).toBeTruthy() // Focus keeps the help open.
  await user.keyboard('{Escape}')
  expect(screen.queryByRole('tooltip')).toBeNull()
  expect(document.activeElement).toBe(help)
  await user.tab()
  await user.tab({ shift: true })
  expect(screen.getByRole('tooltip')).toBeTruthy()
  await user.tab()
  expect(screen.queryByRole('tooltip')).toBeNull()
  expect(choice.checked).toBe(false)
  expect(vi.mocked(fetch).mock.calls.some(([, init]) => init?.method === 'POST')).toBe(false)
})

test('confirmed decision locks controls, retains exact uncertain retry and refreshes committed standing', async () => {
  let committed = false
  let attempts = 0
  let release!: (r: Response) => void
  const writes: RequestInit[] = []
  fixture((path, init) => {
    if (init?.method === 'POST') {
      writes.push(init); attempts++
      if (attempts === 1) return new Promise<Response>(resolve => { release = resolve })
      committed = true
      return response({ caseReference: reference, revision: 1 })
    }
    if (path.endsWith('/cases/' + reference) && committed) return response({ ...initial, revision: 1, state: 'resolved', standing: { ...initial.standing, activeStrikeCount: 1 } })
  })
  const user = await openCase()
  await prepareDecision(user)
  expect(screen.getByRole('dialog')).toBeTruthy()
  expect(screen.getByText(/one strike per case/i)).toBeTruthy()
  await user.click(screen.getByRole('button', { name: 'Confirm decision' }))
  await waitFor(() => expect(writes).toHaveLength(1))
  expect(screen.getByRole('button', { name: 'Submitting…' }).hasAttribute('disabled')).toBe(true)
  expect(screen.getByLabelText('Internal evidence notes').matches(':disabled')).toBe(true)
  release(response({ error: 'unavailable' }, 503))
  await user.click(await screen.findByRole('button', { name: 'Retry exact command' }))
  await screen.findByText(/Command committed/)
  expect(writes).toHaveLength(2)
  expect(writes[1].body).toBe(writes[0].body)
  expect((writes[1].headers as Record<string, string>)['Idempotency-Key']).toBe((writes[0].headers as Record<string, string>)['Idempotency-Key'])
  expect(JSON.parse(String(writes[0].body))).toMatchObject({ expectedRevision: 0, reason: 'spam', consequences: ['visibilityHide', 'strike'], evidenceNotes: 'Reviewed the synthetic evidence.', userSafeDetail: 'Repeated unsolicited promotion.' })
  expect(await screen.findByText('1 of 3')).toBeTruthy()
})

test('committed command remains successful when its read refresh fails', async () => {
  let committed = false
  let failRefresh = true
  fixture((path, init) => {
    if (init?.method === 'POST') { committed = true; return response({ revision: 1 }) }
    if (path.endsWith('/cases/' + reference) && committed) return failRefresh ? response({ error: 'unavailable' }, 503) : response({ ...initial, state: 'resolved', revision: 1 })
  })
  const user = await openCase()
  await user.click(screen.getByRole('button', { name: 'Review decision' }))
  await user.click(screen.getByRole('button', { name: 'Confirm decision' }))
  expect(await screen.findByText('Command committed. Refresh the case before taking another action.')).toBeTruthy()
  expect(await screen.findByText(/Case data could not be refreshed/)).toBeTruthy()
  expect(screen.queryByRole('button', { name: 'Retry exact command' })).toBeNull()
  expect(screen.getByRole('button', { name: 'Review decision' }).matches(':disabled')).toBe(true)
  failRefresh = false
  await user.click(screen.getByRole('button', { name: 'Refresh case' }))
  await screen.findByText('Resolved · revision 1')
})

test('appeal confirmation, changed outcome and effect reversal use separate confirmed commands', async () => {
  let detail = { ...initial, state: 'resolved', revision: 1, appealStatus: '', decisions: [{ disposition: 'violation', reason: 'spam', createdAt: initial.createdAt }], effects: [{ effectId: 'effect-1', type: 'strike', action: 'apply', active: true, createdAt: initial.createdAt }] }
  const writes: { path: string; body: Record<string, unknown> }[] = []
  fixture((path, init) => {
    if (init?.method === 'POST') {
      writes.push({ path, body: JSON.parse(String(init.body)) })
      detail = { ...detail, revision: detail.revision + 1, appealStatus: path.endsWith('/appeal-confirmations') ? 'pending' : 'changed' }
      return response({ revision: detail.revision })
    }
    if (path.endsWith('/cases/' + reference)) return response(detail)
  })
  const user = await openCase()
  await user.click(screen.getByRole('button', { name: 'Review appeal confirmation' }))
  await user.click(screen.getByRole('button', { name: 'Confirm appeal confirmation' }))
  await user.selectOptions(await screen.findByLabelText('Appeal outcome'), 'changed')
  await user.click(screen.getByLabelText('Account strike'))
  await user.type(screen.getByLabelText('Appeal rationale'), 'The evidence supports reversing the strike.')
  await user.click(screen.getByRole('button', { name: 'Review appeal resolution' }))
  await user.click(screen.getByRole('button', { name: 'Confirm appeal resolution' }))
  await waitFor(() => expect(writes).toHaveLength(2))
  expect(writes[0]).toMatchObject({ path: expect.stringContaining('/appeal-confirmations'), body: { expectedRevision: 1 } })
  expect(writes[1]).toMatchObject({ path: expect.stringContaining('/appeal-resolutions'), body: { expectedRevision: 2, outcome: 'changed', reversedEffects: ['strike'], rationale: 'The evidence supports reversing the strike.' } })
})

test('severe suspension restoration identifies its exact active effect and needs its own confirmation', async () => {
  const effectId = '50000000-0000-4000-8000-000000000099'
  const writes: RequestInit[] = []
  fixture((path, init) => {
    if (init?.method === 'POST') { writes.push(init); return response({ revision: 4 }) }
    if (path.endsWith('/cases/' + reference)) return response({ ...initial, state: 'resolved', revision: 3, effects: [{ effectId, type: 'severeSuspension', action: 'apply', active: true, createdAt: initial.createdAt }], standing: { ...initial.standing, severeSuspended: true, suspended: true } })
  })
  const user = await openCase()
  await user.type(screen.getByLabelText('Restoration rationale'), 'Independent review supports restoration.')
  await user.click(screen.getByRole('button', { name: 'Review severe suspension restoration' }))
  expect(writes).toHaveLength(0)
  await user.click(screen.getByRole('button', { name: 'Confirm severe suspension restoration' }))
  await waitFor(() => expect(writes).toHaveLength(1))
  expect(JSON.parse(String(writes[0].body))).toEqual({ expectedRevision: 3, effectId, rationale: 'Independent review supports restoration.' })
})

test('effect changes can reverse and reapply historical effects without conflating severe restoration', async () => {
  const writes: RequestInit[] = []
  fixture((path, init) => {
    if (init?.method === 'POST') { writes.push(init); return response({ revision: 2 }) }
    if (path.endsWith('/cases/' + reference)) return response({ ...initial, state: 'resolved', revision: 1, effects: [
      { effectId: 'hide', type: 'visibilityHide', action: 'apply', active: false, createdAt: initial.createdAt },
      { effectId: 'strike', type: 'strike', action: 'apply', active: true, createdAt: initial.createdAt },
    ] })
  })
  const user = await openCase()
  await user.click(screen.getByLabelText('Reverse Account strike'))
  await user.click(screen.getByLabelText('Reapply Hide content'))
  await user.type(screen.getByLabelText('Effect change rationale'), 'Strike overturned; earlier visibility restriction still applies.')
  await user.click(screen.getByRole('button', { name: 'Review effect change' }))
  await user.click(screen.getByRole('button', { name: 'Confirm effect change' }))
  await waitFor(() => expect(writes).toHaveLength(1))
  expect(JSON.parse(String(writes[0].body))).toMatchObject({ expectedRevision: 1, negateEffects: ['strike'], applyEffects: ['visibilityHide'] })
})

test('direct link and restricted preview expose no content or mutation controls to a read-only operator', async () => {
  window.history.replaceState(null, '', '#/cases/' + reference)
  fixture(path => path.includes('/preview') ? response({ state: 'restricted', text: 'MUST NEVER RENDER', mediaAvailable: false }) : undefined)
  render(<CasesWorkspace operator={{ ...operator, permissions: { 'moderation.read': true, 'moderation.manage': false } }} onSessionExpired={() => {}} />)
  await screen.findByText('Restricted content. Use the specialist review workflow.')
  expect(screen.queryByText('MUST NEVER RENDER')).toBeNull()
  expect(screen.queryByRole('button', { name: 'Review decision' })).toBeNull()
})

test('helpers never request the global queue, including through a direct case link', async () => {
  fixture()
  window.history.replaceState(null, '', '#/cases/' + reference)
  render(<CasesWorkspace operator={{ ...operator, permissionSet: 'helper', permissions: {} }} onSessionExpired={() => {}} />)
  await screen.findByRole('heading', { name: 'Assigned safety work' })
  expect(fetch).not.toHaveBeenCalled()
})

test('late queue results cannot replace a newer filtered queue', async () => {
  let old!: (r: Response) => void
  fixture(path => {
    if (path.includes('/cases?') && path.includes('state=open')) return new Promise<Response>(resolve => { old = resolve })
    if (path.includes('/cases?')) return response({ items: [] })
  })
  const user = userEvent.setup()
  render(<CasesWorkspace operator={operator} onSessionExpired={() => {}} />)
  await user.selectOptions(screen.getByLabelText('Case status'), 'resolved')
  await screen.findByText('No cases match these filters.')
  await act(async () => { old(response({ items: [initial] })) })
  expect(screen.queryByRole('link', { name: 'Review ' + reference })).toBeNull()
})

test('transport failure keeps navigation locked and retries the same reviewed body after hash navigation', async () => {
  const writes: RequestInit[] = []
  fixture((_path, init) => {
    if (init?.method === 'POST') {
      writes.push(init)
      if (writes.length === 1) return Promise.reject(new TypeError('Network unavailable'))
      return response({ revision: 1 })
    }
  })
  const user = await openCase()
  await user.click(screen.getByRole('button', { name: 'Review decision' }))
  await user.click(screen.getByRole('button', { name: 'Confirm decision' }))
  await screen.findByRole('button', { name: 'Retry exact command' })
  expect(screen.getByLabelText('Case status').matches(':disabled')).toBe(true)
  window.location.hash = '#/cases/MOD-50000000-0000-4000-8000-000000000002'
  await waitFor(() => expect(window.location.hash).toBe('#/cases/' + reference))
  await user.click(screen.getByRole('button', { name: 'Retry exact command' }))
  await screen.findByText(/Command committed/)
  expect(writes[0].body).toBe(writes[1].body)
  expect((writes[0].headers as Record<string, string>)['Idempotency-Key']).toBe((writes[1].headers as Record<string, string>)['Idempotency-Key'])
})

test('review can be cancelled without issuing a command and restores keyboard focus', async () => {
  fixture()
  const user = await openCase()
  const button = screen.getByRole('button', { name: 'Review decision' })
  await user.click(button)
  expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Back to review' }))
  await user.tab({ shift: true })
  expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Confirm decision' }))
  await user.tab()
  expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Back to review' }))
  await user.click(screen.getByRole('button', { name: 'Back to review' }))
  expect(screen.queryByRole('dialog')).toBeNull()
  expect(document.activeElement).toBe(button)
  expect(vi.mocked(fetch).mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(0)
})

test('revoked access clears the workspace through the authentication boundary', async () => {
  const expired = vi.fn()
  fixture(path => path.includes('/cases?') ? response({ error: 'operator_access_denied' }, 403) : undefined)
  render(<CasesWorkspace operator={operator} onSessionExpired={expired} />)
  await waitFor(() => expect(expired).toHaveBeenCalled())
})

test('operator can read older provenance through opaque cursor pagination', async () => {
  fixture(path => path.includes('/provenance') ? response({ items: path.includes('cursor=older') ? [{ id: 'older', actorId: 'operator:earlier', sourceSystem: 'craftsky-console', reason: 'Earlier provenance entry', createdAt: initial.createdAt }] : [], ...(path.includes('cursor=older') ? {} : { cursor: 'older' }) }) : undefined)
  const user = await openCase()
  await user.click(screen.getByRole('button', { name: 'Load older provenance' }))
  expect(await screen.findByText('Earlier provenance entry')).toBeTruthy()
})

test('upheld appeal rationale remains readable in case history', async () => {
  fixture(path => path.endsWith('/cases/' + reference) ? response({ ...initial, state: 'resolved', appealStatus: 'upheld', appealRationale: 'Independent review supports the original decision.' }) : undefined)
  await openCase()
  expect(screen.getByText('Independent review supports the original decision.')).toBeTruthy()
})

test('revision conflict fetches fresh case and requires confirmation with a new key', async () => {
  let conflicted = false
  const writes: RequestInit[] = []
  fixture((path, init) => {
    if (init?.method === 'POST') { writes.push(init); conflicted = true; return writes.length === 1 ? response({ error: 'case_revision_conflict' }, 409) : response({ revision: 2 }) }
    if (path.endsWith('/cases/' + reference) && conflicted) return response({ ...initial, revision: 1 })
  })
  const user = await openCase()
  await user.click(screen.getByRole('button', { name: 'Review decision' }))
  await user.click(screen.getByRole('button', { name: 'Confirm decision' }))
  await screen.findByText(/Case changed/)
  await screen.findByText('Open · revision 1')
  expect(writes).toHaveLength(1)
  await user.click(screen.getByRole('button', { name: 'Review decision' }))
  await user.click(screen.getByRole('button', { name: 'Confirm decision' }))
  await screen.findByText(/Command committed/)
  expect(JSON.parse(String(writes[1].body)).expectedRevision).toBe(1)
  expect((writes[1].headers as Record<string, string>)['Idempotency-Key']).not.toBe((writes[0].headers as Record<string, string>)['Idempotency-Key'])
})

test('case content preview renders the post author, images and an accessible full size view', async () => {
  fixture(path => path.includes('/preview') ? response({ state: 'available', subjectType: 'post', subjectDid: 'did:plc:author', text: 'A synthetic craft project.', mediaAvailable: true, author: { did: 'did:plc:author', displayName: 'Mira Makes' }, images: [{ fullsize: 'https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:author/scarf@jpeg', thumb: 'https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:author/scarf@jpeg', alt: 'Blue knitted scarf' }] }) : undefined)
  const user = await openCase()
  expect(screen.getByText('Mira Makes')).toBeTruthy()
  expect(screen.getByRole('img', { name: 'Blue knitted scarf' }).getAttribute('src')).toContain('feed_fullsize')
  await user.click(screen.getByRole('button', { name: 'Enlarge image 1' }))
  expect(screen.getByRole('dialog', { name: 'Image 1' })).toBeTruthy()
  await user.keyboard('{Escape}')
  expect(screen.queryByRole('dialog', { name: 'Image 1' })).toBeNull()
  expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Enlarge image 1' }))
})

test('account and event previews display profile media and event context', async () => {
  let kind = 'account'
  fixture(path => path.includes('/preview') ? response(kind === 'account' ? { state: 'available', subjectType: 'account', subjectDid: 'did:plc:author', title: 'Mira Makes', text: 'A synthetic craft project.', mediaAvailable: true, author: { did: 'did:plc:author', displayName: 'Mira Makes', avatar: 'https://cdn.bsky.app/img/avatar/plain/did:plc:author/avatar@jpeg' }, banner: 'https://cdn.bsky.app/img/banner/plain/did:plc:author/banner@jpeg', crafts: ['knitting', 'weaving'] } : { state: 'available', subjectType: 'event', title: 'Textile fair', text: 'Meet local makers', mediaAvailable: true, event: { startsAt: '2030-01-01T10:00:00Z', endsAt: '2030-01-01T18:00:00Z', venueName: 'Village hall', eventUri: 'https://example.com/fair' }, images: [{ fullsize: 'https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:author/fair@jpeg', alt: 'Textile stalls' }] }) : undefined)
  const user = await openCase()
  expect(screen.getByRole('img', { name: 'Account avatar' })).toBeTruthy()
  expect(screen.getByRole('img', { name: 'Account banner' })).toBeTruthy()
  expect(screen.getByText('Knitting')).toBeTruthy()
  kind = 'event'
  await user.click(screen.getByRole('button', { name: 'Refresh case' }))
  expect(await screen.findByText('Village hall')).toBeTruthy()
  expect(screen.getByRole('img', { name: 'Textile stalls' })).toBeTruthy()
  expect(screen.getByRole('link', { name: 'Event website' }).getAttribute('rel')).toContain('noreferrer')
})

test('post video is only loaded after the operator asks to play it, and restricted previews never render media', async () => {
  let restricted = false
  fixture(path => path.includes('/preview') ? response({ state: restricted ? 'restricted' : 'available', subjectType: 'post', text: 'A synthetic craft project.', mediaAvailable: true, video: { playlist: 'https://video.bsky.app/watch/did:plc:author/video/playlist.m3u8', thumbnail: 'https://video.bsky.app/watch/did:plc:author/video/thumbnail.jpg', alt: 'Weaving demonstration' }, images: restricted ? [{ fullsize: 'https://cdn.bsky.app/never-load', alt: 'Restricted image canary' }] : [] }) : undefined)
  const user = await openCase()
  expect(screen.queryByLabelText('Video preview')).toBeNull()
  await user.click(screen.getByRole('button', { name: 'Load video' }))
  const video = screen.getByLabelText('Video preview') as HTMLVideoElement
  expect(video.controls).toBe(true)
  expect(video.autoplay).toBe(false)
  restricted = true
  await user.click(screen.getByRole('button', { name: 'Refresh case' }))
  expect(await screen.findByText('Restricted content. Use the specialist review workflow.')).toBeTruthy()
  expect(screen.queryByLabelText('Video preview')).toBeNull()
  expect(screen.queryByRole('img', { name: 'Restricted image canary' })).toBeNull()
})


test('post previews show project and link context and handle missing images without unsafe destinations', async () => {
  fixture(path => path.includes('/preview') ? response({ state: 'available', subjectType: 'post', text: 'A synthetic craft project.', mediaAvailable: true, images: [{ fullsize: 'https://cdn.bsky.app/missing', alt: 'Missing scarf photo' }], project: { common: { title: 'Winter scarf', craftType: 'social.craftsky.feed.defs#knitting', materials: [{ text: 'Blue wool' }] } }, external: { title: 'Pattern notes', description: 'A useful pattern', uri: 'javascript:alert(1)' } }) : undefined)
  await openCase()
  expect(screen.getByRole('heading', { name: 'Winter scarf' })).toBeTruthy()
  expect(screen.getByText('Materials: Blue wool')).toBeTruthy()
  expect(screen.getByText('Pattern notes')).toBeTruthy()
  expect(screen.queryByRole('link', { name: 'Open linked page' })).toBeNull()
  fireEvent.error(screen.getByRole('img', { name: 'Missing scarf photo' }))
  expect(screen.getByRole('status').textContent).toContain('Image unavailable')
})
