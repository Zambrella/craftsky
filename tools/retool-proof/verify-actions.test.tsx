import assert from 'node:assert/strict'
import test from 'node:test'
import React from 'react'
import { act, create } from 'react-test-renderer'
import { useModerationActions } from './retool-source/frontend/hooks/useModerationActions'

globalThis.IS_REACT_ACT_ENVIRONMENT = true

function deferred() {
  let resolve!: () => void
  const promise = new Promise<void>((done) => { resolve = done })
  return { promise, resolve }
}

async function mount(options: Parameters<typeof useModerationActions>[0]) {
  let actions!: ReturnType<typeof useModerationActions>
  function Probe() {
    actions = useModerationActions(options)
    return null
  }
  let renderer!: ReturnType<typeof create>
  await act(async () => { renderer = create(<Probe />) })
  return { current: () => actions, close: async () => { await act(async () => { renderer.unmount() }) } }
}

test('ambiguous timeout retries the original body after displayed revision changes and blocks during refresh', async () => {
  const calls: { idempotencyKey: string; expectedRevision: number }[] = []
  let reverseCalls = 0
  const refresh = deferred()
  const harness = await mount({
    runApply: async (command) => {
      calls.push({ ...command })
      if (calls.length === 1) throw new Error('timeout after possible commit')
      return { ok: true, caseReference: 'synthetic', revision: 1, replayed: true }
    },
    runReverse: async () => {
      reverseCalls++
      return { ok: true, caseReference: 'synthetic', revision: 2, replayed: false }
    },
    onSettledSuccess: () => refresh.promise,
  })
  try {
    await act(async () => { await harness.current().execute('apply', 0) })
    assert.equal(harness.current().states.apply.phase, 'error')
    let pending!: Promise<void>
    await act(async () => {
      pending = harness.current().execute('apply', 99)
      await Promise.resolve()
    })
    assert.deepEqual(calls[1], calls[0])
    assert.equal(calls[1].expectedRevision, 0)
    assert.equal(harness.current().states.apply.phase, 'success')
    assert.equal(harness.current().anyRunning, true)
    await act(async () => { await harness.current().execute('reverse', 1) })
    assert.equal(reverseCalls, 0)
    await act(async () => { refresh.resolve(); await pending })
    assert.equal(harness.current().anyRunning, false)
    assert.equal(harness.current().states.apply.replayed, true)
  } finally {
    refresh.resolve()
    await harness.close()
  }
})

test('failed follow-up reads cannot turn a committed command into a retryable failure', async () => {
  const harness = await mount({
    runApply: async () => ({ ok: true, caseReference: 'synthetic', revision: 1, replayed: false }),
    runReverse: async () => ({ ok: true, caseReference: 'synthetic', revision: 2, replayed: false }),
    onSettledSuccess: async () => { throw new Error('read refresh unavailable') },
  })
  try {
    await act(async () => { await harness.current().execute('apply', 0) })
    assert.equal(harness.current().states.apply.phase, 'success')
    assert.equal(harness.current().states.apply.command, null)
    assert.equal(harness.current().states.apply.successRevision, 1)
    assert.equal(harness.current().anyRunning, false)
  } finally {
    await harness.close()
  }
})

test('revision conflict retires the command and fresh review uses a new key and revision', async () => {
  const calls: { idempotencyKey: string; expectedRevision: number }[] = []
  const harness = await mount({
    runApply: async (command) => {
      calls.push({ ...command })
      if (calls.length === 1) return { ok: false, conflict: true, code: 'case_revision_conflict', requestId: 'synthetic-request', message: 'case revision conflict' }
      return { ok: true, caseReference: 'synthetic', revision: 3, replayed: false }
    },
    runReverse: async () => ({ ok: true, caseReference: 'synthetic', revision: 4, replayed: false }),
    onSettledSuccess: async () => {},
  })
  try {
    await act(async () => { await harness.current().execute('apply', 0) })
    assert.equal(harness.current().states.apply.phase, 'conflict')
    assert.equal(harness.current().states.apply.command, null)
    await act(async () => { harness.current().dismiss('apply') })
    await act(async () => { await harness.current().execute('apply', 2) })
    assert.equal(calls[1].expectedRevision, 2)
    assert.notEqual(calls[1].idempotencyKey, calls[0].idempotencyKey)
  } finally {
    await harness.close()
  }
})
