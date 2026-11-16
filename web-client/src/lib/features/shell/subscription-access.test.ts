import { afterEach, describe, expect, it, vi } from 'vitest';
import type { SubscriptionAccess } from '#lib/api/types.ts';
import { SubscriptionAccessState } from './subscription-access.svelte.ts';

const did = 'did:plc:alice';
const account = `session-a:${did}`;
const access = (tier: 'free' | 'plus' | 'business'): SubscriptionAccess => ({
	did,
	effectiveTier: tier,
	givesAccess: tier !== 'free'
});
afterEach(() => vi.useRealTimers());

describe('shared subscription access', () => {
	it('refreshes on a crossed access boundary or age without revoking provider-confirmed grace access', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-11-16T10:00:00Z'));
		const state = new SubscriptionAccessState(account, did, () => account);
		expect(state.needsRefresh()).toBe(true);
		const value = {
			...access('business'),
			accessEndsAt: '2026-11-16T10:01:00Z'
		};
		state.beginRead(account)(value);
		expect(state.needsRefresh()).toBe(false);
		vi.advanceTimersByTime(60_000);
		expect(state.needsRefresh()).toBe(true);
		expect(state.value).toEqual(value);
		state.beginRead(account)(value);
		expect(state.needsRefresh()).toBe(false);
		vi.advanceTimersByTime(5 * 60_000);
		expect(state.needsRefresh()).toBe(true);
		state.beginRead(account)(access('free'));
		expect(state.value).toEqual(access('free'));
		expect(state.needsRefresh()).toBe(false);
	});
	it('publishes both newly paid access and confirmed access loss', () => {
		const state = new SubscriptionAccessState(account, did, () => account);
		for (const tier of ['free', 'plus', 'business', 'free'] as const) {
			expect(state.beginRead(account)(access(tier))).toBe(true);
			expect(state.value).toEqual(access(tier));
		}
	});
	it('rejects an earlier boot or refresh after a newer read has begun', () => {
		const state = new SubscriptionAccessState(account, did, () => account);
		const boot = state.beginRead(account);
		const refresh = state.beginRead(account);
		expect(refresh(access('plus'))).toBe(true);
		expect(boot(access('free'))).toBe(false);
		expect(state.value).toEqual(access('plus'));
	});
	it('rejects another DID or session without replacing confirmed access', () => {
		const state = new SubscriptionAccessState(account, did, () => account);
		state.beginRead(account)(access('plus'));
		expect(
			state.beginRead(account)({ ...access('business'), did: 'did:plc:bob' })
		).toBe(false);
		expect(state.beginRead(`session-b:${did}`)(access('business'))).toBe(false);
		expect(state.value).toEqual(access('plus'));
	});
	it('rejects a response after switching accounts or closing its scope', () => {
		let current = account;
		const previous = new SubscriptionAccessState(account, did, () => current);
		const pending = previous.beginRead(account);
		current = 'session-b:did:plc:bob';
		expect(pending(access('plus'))).toBe(false);
		previous.dispose();
		current = account;
		expect(pending(access('plus'))).toBe(false);
		expect(previous.beginRead(account)(access('business'))).toBe(false);
		expect(previous.value).toBeUndefined();
		const replacement = new SubscriptionAccessState(
			account,
			did,
			() => current
		);
		expect(replacement.beginRead(account)(access('free'))).toBe(true);
		expect(replacement.value).toEqual(access('free'));
	});
});
