import { describe, expect, it, vi } from 'vitest';
import type { BillingState, SubscriptionAccess } from '#lib/api/types.ts';
import { BillingIdentity } from './billing-identity';
import { SubscriptionAccessState } from './subscription-access.svelte';

const did = 'did:plc:alice';
const account = `session-a:${did}`;
const state: BillingState = {
	billingAccountId: 'account',
	revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
	requestedGeneration: 1,
	reconciledGeneration: 0,
	reconciliationStale: true,
	webCheckoutAvailable: false,
	subscriptions: [],
	licenses: []
};
const access = (tier: 'free' | 'plus' = 'free'): SubscriptionAccess => ({
	did,
	effectiveTier: tier,
	givesAccess: tier === 'plus'
});

function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (error: unknown) => void;
	const promise = new Promise<T>((complete, fail) => {
		resolve = complete;
		reject = fail;
	});
	return { promise, resolve, reject };
}

function fixture(current = () => account, allowed = () => true) {
	const shared = new SubscriptionAccessState(account, did, current);
	shared.beginRead(account)(access());
	const billing = {
		ensure: vi.fn(async () => state),
		access: vi.fn(async () => access('plus'))
	};
	const report = vi.fn<(error: unknown) => void | Promise<void>>();
	const identity = new BillingIdentity(
		account,
		current,
		billing,
		shared,
		allowed,
		report
	);
	return { billing, identity, shared, report };
}

describe('active browser billing identity', () => {
	it('ensures the authenticated account before publishing refreshed access', async () => {
		const { billing, identity, shared } = fixture();
		const ready = deferred<BillingState>();
		billing.ensure.mockReturnValue(ready.promise);
		const first = identity.synchronize();
		await identity.synchronize();
		expect(billing.ensure).toHaveBeenCalledOnce();
		expect(billing.access).not.toHaveBeenCalled();
		expect(shared.value).toEqual(access());
		ready.resolve(state);
		await first;
		expect(billing.access).toHaveBeenCalledOnce();
		expect(shared.value).toEqual(access('plus'));
		await identity.synchronize();
		expect(billing.ensure).toHaveBeenCalledOnce();
	});

	it('reports original optional registration and refresh failures once without rejecting', async () => {
		for (const failure of ['ensure', 'access'] as const) {
			const { billing, identity, shared, report } = fixture();
			const error = new Error('setup unavailable');
			billing[failure].mockRejectedValue(error);
			await expect(identity.synchronize()).resolves.toBeUndefined();
			await identity.synchronize();
			expect(report).toHaveBeenCalledExactlyOnceWith(error);
			expect(shared.value).toEqual(access());
			if (failure === 'ensure') expect(billing.access).not.toHaveBeenCalled();
		}
	});

	for (const failure of ['ensure', 'access'] as const)
		for (const fence of ['account', 'disposal', 'readiness'])
			it(`discards a delayed ${failure} failure after ${fence} changes`, async () => {
				let current = account,
					allowed = true;
				const { billing, identity, shared, report } = fixture(
					() => current,
					() => allowed
				);
				const response = deferred<never>();
				billing[failure].mockReturnValue(response.promise);
				const pending = identity.synchronize();
				await vi.waitFor(() => expect(billing[failure]).toHaveBeenCalledOnce());
				if (fence === 'account') current = 'session-b:did:plc:bob';
				else if (fence === 'disposal') identity.dispose();
				else allowed = false;
				response.reject(new Error('late failure'));
				await expect(pending).resolves.toBeUndefined();
				expect(report).not.toHaveBeenCalled();
				expect(shared.value).toEqual(access());
			});

	for (const asynchronous of [false, true])
		it(`contains ${asynchronous ? 'asynchronous' : 'synchronous'} reporting failures`, async () => {
			const { billing, identity, shared, report } = fixture();
			billing.ensure.mockRejectedValue(new Error('setup unavailable'));
			report.mockImplementation(() => {
				const error = new Error('reporter unavailable');
				if (asynchronous) return Promise.reject(error);
				throw error;
			});
			await expect(identity.synchronize()).resolves.toBeUndefined();
			expect(report).toHaveBeenCalledOnce();
			expect(billing.access).not.toHaveBeenCalled();
			expect(shared.value).toEqual(access());
		});

	it('does not dispatch or publish after its account scope is closed', async () => {
		for (const afterEnsure of [false, true]) {
			const { billing, identity, shared } = fixture();
			if (afterEnsure) {
				const ready = deferred<BillingState>();
				billing.ensure.mockReturnValue(ready.promise);
				const pending = identity.synchronize();
				identity.dispose();
				ready.resolve(state);
				await pending;
			} else {
				identity.dispose();
				await identity.synchronize();
				expect(billing.ensure).not.toHaveBeenCalled();
			}
			expect(billing.access).not.toHaveBeenCalled();
			expect(shared.value).toEqual(access());
		}
	});

	it('discards delayed responses after switching accounts', async () => {
		let current = account;
		const { billing, identity, shared } = fixture(() => current);
		const response = deferred<SubscriptionAccess>();
		billing.access.mockReturnValue(response.promise);
		const pending = identity.synchronize();
		await vi.waitFor(() => expect(billing.access).toHaveBeenCalledOnce());
		current = 'session-b:did:plc:bob';
		response.resolve(access('plus'));
		await pending;
		expect(shared.value).toEqual(access());
	});

	it('rejects wrong-DID access and preserves a newer billing refresh', async () => {
		const { billing, identity, shared } = fixture();
		billing.access.mockResolvedValue({ ...access('plus'), did: 'did:plc:bob' });
		await identity.synchronize();
		expect(shared.value).toEqual(access());

		const next = fixture();
		const response = deferred<SubscriptionAccess>();
		next.billing.access.mockReturnValue(response.promise);
		const pending = next.identity.synchronize();
		await vi.waitFor(() => expect(next.billing.access).toHaveBeenCalledOnce());
		next.shared.beginRead(account)(access('plus'));
		response.resolve(access());
		await pending;
		expect(next.shared.value).toEqual(access('plus'));
	});

	it('registers again for a restored session in a new shell', async () => {
		const first = fixture();
		await first.identity.synchronize();
		first.identity.dispose();
		const restored = fixture();
		await restored.identity.synchronize();
		expect(first.billing.ensure).toHaveBeenCalledOnce();
		expect(restored.billing.ensure).toHaveBeenCalledOnce();
		expect(restored.shared.value).toEqual(access('plus'));
	});

	it('does not supersede a user refresh that starts while registration is pending', async () => {
		const { billing, identity, shared } = fixture();
		const ready = deferred<BillingState>();
		billing.ensure.mockReturnValue(ready.promise);
		billing.access.mockResolvedValue(access());
		const pending = identity.synchronize();
		const userRefresh = shared.beginRead(account);
		ready.resolve(state);
		await pending;
		expect(userRefresh(access('plus'))).toBe(true);
		expect(shared.value).toEqual(access('plus'));
	});

	it('respects readiness changes before and during optional registration', async () => {
		let allowed = false;
		const { billing, identity, shared } = fixture(
			() => account,
			() => allowed
		);
		await identity.synchronize();
		expect(billing.ensure).not.toHaveBeenCalled();
		allowed = true;
		const ready = deferred<BillingState>();
		billing.ensure.mockReturnValue(ready.promise);
		const pending = identity.synchronize();
		allowed = false;
		ready.resolve(state);
		await pending;
		expect(billing.ensure).toHaveBeenCalledOnce();
		expect(billing.access).not.toHaveBeenCalled();
		expect(shared.value).toEqual(access());
	});
});
