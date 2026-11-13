import { describe, expect, it, vi } from 'vitest';
import type { BillingState, BillingSubscription } from '#lib/api/types.ts';
import {
	canStartWebCheckout,
	hasCurrentSubscription,
	hostedCheckoutURL,
	SubscriptionCheckout,
	SubscriptionManagement,
	stripePortalURL,
	subscriptionProvider,
	subscriptionNeedsRefresh
} from './subscription-checkout.ts';

const privateId = '20000000-0000-4000-8000-000000000001';
const checkoutURL =
	'https://checkout.stripe.com/c/pay/cs_test_fixture#opaque%2fcheckout%3Fcapability';
const portalURL =
	'https://billing.stripe.com/p/session/test_portal#opaque%2fportal';
const state = (): BillingState => ({
	billingAccountId: 'account',
	revenueCatAppUserId: privateId,
	requestedGeneration: 1,
	reconciledGeneration: 1,
	reconciliationStale: false,
	reconciledAt: new Date().toISOString(),
	webCheckoutAvailable: true,
	subscriptions: [],
	licenses: []
});
const subscription = (
	changes: Partial<BillingSubscription> = {}
): BillingSubscription => ({
	id: 'subscription',
	productId: 'plus',
	store: 'stripe',
	status: 'active',
	givesAccess: true,
	pendingPayment: false,
	anomaly: 'none',
	...changes
});

describe('account subscription checkout eligibility', () => {
	it('allows a retained checkout to be resumed while AppView enforces its tier', () => {
		for (const webCheckoutPending of [undefined, false, true]) {
			const billing = { ...state(), webCheckoutPending };
			expect(canStartWebCheckout(billing)).toBe(true);
			expect(
				canStartWebCheckout({ ...billing, reconciliationStale: true })
			).toBe(false);
			expect(
				canStartWebCheckout({ ...billing, subscriptions: [subscription()] })
			).toBe(false);
		}
	});
	it('offers a choice only after provider confirmation and when configured', () => {
		expect(canStartWebCheckout(state())).toBe(true);
		for (const changes of [
			{ webCheckoutAvailable: undefined },
			{ webCheckoutAvailable: false },
			{ reconciledAt: undefined },
			{ reconciledAt: 'not-a-date' },
			{ reconciledAt: new Date(Date.now() - 301_000).toISOString() },
			{ reconciledAt: new Date(Date.now() + 61_000).toISOString() },
			{ reconciliationStale: true },
			{ requestedGeneration: 2 }
		])
			expect(canStartWebCheckout({ ...state(), ...changes })).toBe(false);
	});
	it('blocks native and Stripe subscriptions, pending, canceled, paused and unknown states', () => {
		for (const changes of [
			{},
			{ store: 'app_store' },
			{ store: 'play_store' },
			{ pendingPayment: true, givesAccess: false },
			{ status: 'cancelled', autoRenewalStatus: 'will_not_renew' },
			{ status: 'paused', givesAccess: false },
			{ status: 'future_status', givesAccess: false }
		]) {
			const value = { ...state(), subscriptions: [subscription(changes)] };
			expect(hasCurrentSubscription(value)).toBe(true);
			expect(canStartWebCheckout(value)).toBe(false);
		}
	});
	it('permits another tier only after terminal state and paid-through time end', () => {
		for (const status of ['expired', 'refunded', 'revoked']) {
			const ended = subscription({
				status,
				givesAccess: false,
				currentPeriodEndsAt: new Date(Date.now() - 1000).toISOString()
			});
			expect(canStartWebCheckout({ ...state(), subscriptions: [ended] })).toBe(
				true
			);
			expect(
				canStartWebCheckout({
					...state(),
					subscriptions: [
						{
							...ended,
							currentPeriodEndsAt: new Date(Date.now() + 60_000).toISOString()
						}
					]
				})
			).toBe(false);
		}
	});
	it('never offers checkout for a terminal subscription with unresolved anomalies', () => {
		const value = {
			...state(),
			subscriptions: [
				subscription({
					status: 'expired',
					givesAccess: false,
					anomaly: 'ownership_conflict'
				})
			]
		};
		expect(canStartWebCheckout(value)).toBe(false);
		expect(subscriptionNeedsRefresh(state())).toBe(false);
	});
});

describe('Stripe checkout isolation', () => {
	it('passes only the chosen tier; AppView supplies the customer identity', async () => {
		const request = vi.fn(async () => ({ url: checkoutURL }));
		const navigate = vi.fn();
		const checkout = new SubscriptionCheckout(
			'account-a',
			() => 'account-a',
			request,
			navigate
		);
		await checkout.start('business');
		expect(request).toHaveBeenCalledExactlyOnceWith('business');
		expect(navigate).toHaveBeenCalledExactlyOnceWith(checkoutURL);
	});
	it('prevents duplicate requests and discards a response after an account switch', async () => {
		let current = 'account-a';
		let finish!: (value: { url: string }) => void;
		const request = vi.fn(
			() =>
				new Promise<{ url: string }>((resolve) => {
					finish = resolve;
				})
		);
		const navigate = vi.fn();
		const checkout = new SubscriptionCheckout(
			'account-a',
			() => current,
			request,
			navigate
		);
		const first = checkout.start('plus');
		await checkout.start('business');
		current = 'account-b';
		finish({ url: checkoutURL });
		await first;
		expect(request).toHaveBeenCalledTimes(1);
		expect(navigate).not.toHaveBeenCalled();
	});
	it('cannot navigate after closing the view and can retry a failed request', async () => {
		const request = vi
			.fn()
			.mockRejectedValueOnce(new Error('network'))
			.mockResolvedValue({ url: checkoutURL });
		const navigate = vi.fn();
		const checkout = new SubscriptionCheckout(
			'account-a',
			() => 'account-a',
			request,
			navigate
		);
		await expect(checkout.start('plus')).rejects.toThrow('network');
		await checkout.start('plus');
		checkout.dispose();
		await checkout.start('business');
		expect(request).toHaveBeenCalledTimes(2);
		expect(navigate).toHaveBeenCalledTimes(1);
	});
	it('preserves valid Stripe checkout capabilities verbatim', () => {
		for (const value of [
			checkoutURL,
			'https://checkout.stripe.com/c/pay/cs_live_abc123',
			'https://checkout.stripe.com/c/pay/cs_test_abc123#' + 'x'.repeat(800)
		])
			expect(hostedCheckoutURL(value)).toBe(value);
	});
	it('rejects unsafe checkout URLs without exposing the private response', () => {
		for (const url of [
			`http://checkout.stripe.com/c/pay/cs_test_${privateId}`,
			'https://checkout.stripe.com.evil.test/c/pay/cs_test_secret',
			'https://checkout.stripe.com@evil.test/c/pay/cs_test_secret',
			'https://user@checkout.stripe.com/c/pay/cs_test_secret',
			'https://checkout.stripe.com:8443/c/pay/cs_test_secret',
			'https://checkout.stripe.com:443/c/pay/cs_test_secret',
			'https://checkout.stripe.com/c/pay/cs_test_secret?redirect=evil',
			'https://checkout.stripe.com/c/pay/cs_test_secret/extra',
			'https://checkout.stripe.com/c/pay/cs_test_',
			'https://checkout.stripe.com/c/pay/cs_test_a%2fb',
			'https://checkout.stripe.com/c/pay/../pay/cs_test_secret',
			'https://checkout.stripe.com/c/pay/cs_test_secret#private\nCANARY',
			'https://checkout.stripe.com/c/pay/cs_test_secret\n',
			`https://pay.rev.cat/link/${privateId}`,
			'javascript:alert(1)',
			'invalid'
		]) {
			expect(() => hostedCheckoutURL(url)).toThrow(
				'Checkout is unavailable. Please try again later.'
			);
			try {
				hostedCheckoutURL(url);
			} catch (error) {
				expect(String(error)).not.toContain(privateId);
				expect(String(error)).not.toContain('secret');
				expect(String(error)).not.toContain('CANARY');
			}
		}
	});
});

describe('account-owned Stripe subscription management', () => {
	it('opens only a customer-specific portal session and preserves its fragment', () => {
		expect(stripePortalURL(portalURL)).toBe(portalURL);
		expect(subscriptionProvider('stripe')).toBe('Stripe');
		for (const url of [
			'https://billing.stripe.com/p/login/test_sharedEmail',
			'https://billing.stripe.com/p/session/',
			'https://billing.stripe.com/p/session/test_portal?customer=other',
			'https://billing.stripe.com/p/session/test_portal/other',
			'https://billing.stripe.com/p/session/test_portal#private\nCANARY',
			'https://billing.stripe.com/p/session/test_portal\n',
			'https://billing.stripe.com:443/p/session/test_portal',
			'https://user@billing.stripe.com/p/session/test_portal',
			'https://billing.stripe.com.evil.test/p/session/test_portal',
			'https://evil.test/p/session/test_portal',
			checkoutURL
		]) {
			expect(() => stripePortalURL(url)).toThrow(
				'Subscription management is unavailable. Please try again later.'
			);
		}
	});
	it('requests no customer identity and prevents duplicate or switched-account navigation', async () => {
		let current = 'account-a';
		let finish!: (value: { url: string }) => void;
		const request = vi.fn(
			() => new Promise<{ url: string }>((resolve) => (finish = resolve))
		);
		const navigate = vi.fn();
		const management = new SubscriptionManagement(
			'account-a',
			() => current,
			request,
			navigate
		);
		const first = management.start();
		await management.start();
		current = 'account-b';
		finish({ url: portalURL });
		await first;
		expect(request).toHaveBeenCalledExactlyOnceWith();
		expect(navigate).not.toHaveBeenCalled();
	});
	it('discards a pending response after disposal and retries safely after failure', async () => {
		let finish!: (value: { url: string }) => void;
		const request = vi
			.fn()
			.mockRejectedValueOnce(new Error('network'))
			.mockResolvedValueOnce({ url: portalURL })
			.mockImplementationOnce(
				() => new Promise<{ url: string }>((resolve) => (finish = resolve))
			);
		const navigate = vi.fn();
		const management = new SubscriptionManagement(
			'account-a',
			() => 'account-a',
			request,
			navigate
		);
		await expect(management.start()).rejects.toThrow('network');
		await management.start();
		expect(navigate).toHaveBeenCalledExactlyOnceWith(portalURL);
		const pending = management.start();
		management.dispose();
		finish({ url: portalURL });
		await pending;
		expect(navigate).toHaveBeenCalledTimes(1);
	});
});
