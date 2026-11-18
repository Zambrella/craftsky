import { afterEach, expect, it, vi } from 'vitest';
import type { Offering, PurchaseParams } from '@revenuecat/purchases-js';
import { readFileSync } from 'node:fs';
import {
	RevenueCatPaywall,
	type WebPaywallDetails
} from './revenuecat-paywall';

type SDK = typeof import('@revenuecat/purchases-js');
type PaywallParams = Parameters<
	InstanceType<SDK['Purchases']>['presentPaywall']
>[0];
const details: WebPaywallDetails = {
	apiKey: 'strp_sb_fixture',
	appUserId: 'private-owner',
	offering: 'business'
};
const scopes: RevenueCatPaywall[] = [];
afterEach(() => {
	scopes.splice(0).forEach((s) => s.close());
	vi.unstubAllGlobals();
});
function fixture(
	guard = vi.fn(async () => {}),
	current = () => true,
	presentation?: {
		purchaseTarget: HTMLElement;
		onCheckoutChange: (open: boolean) => void;
	}
) {
	let callbacks!: PaywallParams;
	let resolve!: (value: unknown) => void;
	let reject!: (value: unknown) => void;
	const outcome = new Promise((yes, no) => {
		resolve = yes;
		reject = no;
	});
	const purchase = vi.fn<
		(params: PurchaseParams) => Promise<{ customerInfo: object }>
	>(async () => ({ customerInfo: {} }));
	const offering = {
		identifier: 'business',
		availablePackages: [{ identifier: '$rc_monthly' }]
	} as Offering;
	const sdk = {
		purchase,
		close: vi.fn(),
		getOfferings: vi.fn(async () => ({ all: { business: offering } })),
		presentPaywall: vi.fn((params: PaywallParams) => {
			callbacks = params;
			return outcome;
		})
	};
	const configure = vi.fn(() => sdk);
	const load = vi.fn(
		async () =>
			({
				Purchases: { configure, setLogLevel: vi.fn() },
				LogLevel: { Silent: 'silent' }
			}) as unknown as SDK
	);
	const scope = new RevenueCatPaywall(details, guard, current, load);
	scopes.push(scope);
	const ready = vi.fn(),
		completed = vi.fn(),
		canceled = vi.fn(),
		failed = vi.fn();
	const running = scope.present(
		{} as HTMLElement,
		ready,
		completed,
		canceled,
		failed,
		presentation
	);
	return {
		scope,
		sdk,
		purchase,
		configure,
		guard,
		ready,
		completed,
		canceled,
		failed,
		running,
		resolve,
		reject,
		offering,
		callbacks: () => callbacks
	};
}
async function ready(f: ReturnType<typeof fixture>) {
	await vi.waitFor(() => expect(f.ready).toHaveBeenCalledOnce());
}
it('uses the identified account and exact mobile offering; every purchase rechecks the server', async () => {
	const f = fixture();
	await ready(f);
	expect(f.configure).toHaveBeenCalledWith({
		apiKey: details.apiKey,
		appUserId: details.appUserId
	});
	expect(f.callbacks().offering).toBe(f.offering);
	await f.sdk.purchase({} as PurchaseParams);
	await f.sdk.purchase({} as PurchaseParams);
	expect(f.guard).toHaveBeenCalledTimes(2);
	expect(f.purchase).toHaveBeenCalledTimes(2);
	f.resolve({});
	await f.running;
	expect(f.completed).toHaveBeenCalledOnce();
});
it('blocks checkout when fresh server preflight fails and ignores a result after account switch', async () => {
	let current = true;
	const f = fixture(
		vi.fn(async () => {
			throw new Error('stale');
		}),
		() => current
	);
	await ready(f);
	await expect(f.sdk.purchase({} as PurchaseParams)).rejects.toThrow('stale');
	expect(f.purchase).not.toHaveBeenCalled();
	current = false;
	f.resolve({});
	await f.running;
	expect(f.completed).not.toHaveBeenCalled();
});
it('closing while preflight is in flight prevents SDK purchase and releases the next account', async () => {
	let release!: () => void;
	const f = fixture(
		vi.fn(
			() =>
				new Promise<void>((r) => {
					release = r;
				})
		)
	);
	await ready(f);
	const attempt = f.sdk.purchase({} as PurchaseParams);
	f.scope.close();
	release();
	await expect(attempt).rejects.toMatchObject({ name: 'AbortError' });
	await f.running;
	expect(f.purchase).not.toHaveBeenCalled();
	expect(f.sdk.close).toHaveBeenCalledOnce();
	const next = fixture();
	await ready(next);
	f.resolve({});
	expect(f.completed).not.toHaveBeenCalled();
	next.scope.close();
	await next.running;
});
it('treats cancellation as dismissal and prevents concurrent SDK windows', async () => {
	const f = fixture();
	await ready(f);
	const second = fixture();
	await expect(second.running).rejects.toThrow(/already.*open/);
	expect(second.configure).not.toHaveBeenCalled();
	f.reject({ errorCode: 1 });
	await f.running;
	expect(f.canceled).toHaveBeenCalledOnce();
	expect(f.completed).not.toHaveBeenCalled();
});
it('honors the browser cross-tab account lock before loading the SDK', async () => {
	const request = vi.fn(async (_key, _options, callback) => callback(null));
	vi.stubGlobal('navigator', { locks: { request } });
	const f = fixture();
	await expect(f.running).rejects.toThrow('another tab');
	expect(f.configure).not.toHaveBeenCalled();
	expect(request.mock.calls[0]?.[1]).toEqual({ ifAvailable: true });
});
it('pinned SDK paywalls still dispatch purchases through the guarded public method', () => {
	const installed = JSON.parse(
		readFileSync('node_modules/@revenuecat/purchases-js/package.json', 'utf8')
	);
	expect(installed.version).toBe('1.74.0');
	const source = readFileSync(
		'node_modules/@revenuecat/purchases-js/dist/Purchases.es.js',
		'utf8'
	);
	expect(source).toMatch(/await this\.purchase\(/);
});

it('reports retryable SDK errors without exposing provider payloads', async () => {
	const f = fixture();
	await ready(f);
	f.callbacks().listener?.onPurchaseError?.(
		new Error('private-provider-canary')
	);
	expect(f.failed).toHaveBeenCalledWith(
		'Payment could not be completed. Please try again.'
	);
	f.scope.close();
	await f.running;
});

it('keeps paywall and checkout targets separate and restores the sheet after checkout cancellation', async () => {
	const onCheckoutChange = vi.fn();
	const purchaseTarget = {} as HTMLElement;
	const f = fixture(undefined, undefined, { purchaseTarget, onCheckoutChange });
	await ready(f);
	expect(f.callbacks().purchaseHtmlTarget).toBe(purchaseTarget);
	f.purchase.mockRejectedValueOnce({ errorCode: 1 });
	await expect(f.sdk.purchase({} as PurchaseParams)).rejects.toMatchObject({
		errorCode: 1
	});
	expect(onCheckoutChange.mock.calls).toEqual([[true], [false]]);
	f.callbacks().listener?.onPurchaseError?.({
		errorCode: 1
	} as unknown as Error);
	expect(f.failed).not.toHaveBeenCalled();
	f.scope.close();
	await f.running;
});

it('waits for checkout layout before mounting the provider and stops if closed meanwhile', async () => {
	let release!: () => void;
	const expanded = vi.fn(
		() =>
			new Promise<void>((resolve) => {
				release = resolve;
			})
	);
	const f = fixture(undefined, undefined, {
		purchaseTarget: {} as HTMLElement,
		onCheckoutChange: expanded
	});
	await ready(f);
	const attempt = f.sdk.purchase({} as PurchaseParams);
	await vi.waitFor(() => expect(expanded).toHaveBeenCalledWith(true));
	expect(f.purchase).not.toHaveBeenCalled();
	f.scope.close();
	release();
	await expect(attempt).rejects.toMatchObject({ name: 'AbortError' });
	expect(f.purchase).not.toHaveBeenCalled();
	await f.running;
});

it('allows only one purchase through asynchronous preflight and releases the guard after cancellation', async () => {
	let release!: () => void;
	const guard = vi.fn(
		() =>
			new Promise<void>((resolve) => {
				release = resolve;
			})
	);
	const f = fixture(guard);
	await ready(f);
	const first = f.sdk.purchase({} as PurchaseParams);
	await expect(f.sdk.purchase({} as PurchaseParams)).rejects.toThrow(
		'already in progress'
	);
	expect(guard).toHaveBeenCalledOnce();
	f.purchase.mockRejectedValueOnce({ errorCode: 1 });
	release();
	await expect(first).rejects.toMatchObject({ errorCode: 1 });
	guard.mockResolvedValue();
	await f.sdk.purchase({} as PurchaseParams);
	expect(f.purchase).toHaveBeenCalledTimes(2);
	f.scope.close();
	await f.running;
});
