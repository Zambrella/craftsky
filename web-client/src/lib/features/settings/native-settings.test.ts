import { describe, expect, it } from 'vitest';
import type {
	BillingState,
	BillingSubscription,
	BillingLicense
} from '#lib/api/types.ts';
import { imageBody, productBody, type Product } from './model.ts';
import {
	canAssignLicense,
	characterCount,
	currencyOptions,
	effectiveCustomisation,
	followerChart,
	moderationLabel,
	requireRecordText,
	tierState
} from './native-settings.ts';

const license: BillingLicense = {
	id: 'plus-license',
	subscriptionId: 'plus-subscription',
	tier: 'plus',
	assignable: true,
	anomaly: 'none'
};
const subscription: BillingSubscription = {
	id: 'plus-subscription',
	productId: 'craftsky.plus',
	store: 'app_store',
	status: 'active',
	givesAccess: true,
	pendingPayment: false,
	anomaly: 'none'
};
const state: BillingState = {
	billingAccountId: 'billing',
	revenueCatAppUserId: 'billing-owner',
	requestedGeneration: 2,
	reconciledGeneration: 2,
	reconciliationStale: false,
	subscriptions: [subscription],
	licenses: [license]
};

describe('native subscription state precedence', () => {
	it('keeps independent tier cards and permits only authoritative active assignments', () => {
		expect(tierState(state, 'plus')).toBe('active');
		expect(tierState(state, 'business')).toBe('empty');
		expect(canAssignLicense(state, license)).toBe(true);
		expect(canAssignLicense(state, { ...license, assignable: false })).toBe(
			false
		);
		expect(
			tierState(
				{ ...state, subscriptions: [{ ...subscription, status: 'cancelled' }] },
				'plus'
			)
		).toBe('cancelled');
		expect(
			tierState(
				{
					...state,
					subscriptions: [
						{ ...subscription, status: 'expired', givesAccess: false }
					]
				},
				'plus'
			)
		).toBe('dormant');
	});
	it('does not allow assignment for missing, anomalous, stale, pending or future store states', () => {
		const cases: [BillingState, string][] = [
			[{ ...state, subscriptions: [] }, 'anomaly'],
			[
				{ ...state, licenses: [{ ...license, anomaly: 'future-anomaly' }] },
				'anomaly'
			],
			[
				{
					...state,
					subscriptions: [{ ...subscription, anomaly: 'ownership-conflict' }]
				},
				'anomaly'
			],
			[{ ...state, reconciliationStale: true }, 'stale'],
			[{ ...state, requestedGeneration: 3 }, 'stale'],
			[
				{
					...state,
					subscriptions: [{ ...subscription, pendingPayment: true }]
				},
				'pending'
			],
			[
				{
					...state,
					subscriptions: [{ ...subscription, status: 'future-store-state' }]
				},
				'unavailable'
			]
		];
		for (const [value, status] of cases) {
			expect(tierState(value, 'plus')).toBe(status);
			expect(canAssignLicense(value, value.licenses[0])).toBe(false);
		}
	});
	it('uses native anomaly > stale > pending > unknown-status precedence', () => {
		const pending = { ...subscription, pendingPayment: true, status: 'future' };
		expect(
			tierState(
				{ ...state, reconciliationStale: true, subscriptions: [pending] },
				'plus'
			)
		).toBe('stale');
		expect(tierState({ ...state, subscriptions: [pending] }, 'plus')).toBe(
			'pending'
		);
		expect(
			tierState(
				{ ...state, reconciliationStale: true, subscriptions: [] },
				'plus'
			)
		).toBe('anomaly');
		const anomalous = { ...license, id: 'second', subscriptionId: 'missing' };
		expect(
			canAssignLicense({ ...state, licenses: [license, anomalous] }, license)
		).toBe(false);
	});
});

describe('native record text limits', () => {
	it('counts visible characters instead of UTF-16 code units or combining marks', () => {
		expect(characterCount('🧶e\u0301👩‍👩‍👧‍👧')).toBe(3);
		expect(() =>
			requireRecordText('🧶'.repeat(150), 'Title', 150, 1500)
		).not.toThrow();
		expect(() =>
			requireRecordText('🧶'.repeat(151), 'Title', 150, 1500)
		).toThrow('Title is too long');
		expect(() =>
			requireRecordText('👩‍👩‍👧‍👧'.repeat(100), 'Title', 150, 1500)
		).toThrow('Title is too long');
	});
	it('accepts the native 1000-character / 10000-byte image alt range', () => {
		const image = {
			cid: 'bafyimage',
			mime: 'image/jpeg',
			size: 100,
			alt: '🧶'.repeat(1000)
		};
		expect(imageBody(image)?.alt).toBe(image.alt);
		expect(() => imageBody({ ...image, alt: 'a'.repeat(1001) })).toThrow(
			'Alternative text'
		);
		expect(() => imageBody({ ...image, alt: '👩‍👩‍👧‍👧'.repeat(401) })).toThrow(
			'Alternative text'
		);
	});
	it('preserves valid emoji product titles without weakening byte limits', () => {
		const product: Product = {
			title: '🧶'.repeat(150),
			uri: 'https://shop.example/wool',
			amount: '',
			currency: '',
			image: { cid: 'bafyimage', mime: 'image/jpeg', size: 100, alt: '' }
		};
		expect(productBody([product])[0].title).toBe(product.title);
		expect(() =>
			productBody([{ ...product, title: '🧶'.repeat(151) }])
		).toThrow('Product title');
	});
});

describe('native settings catalogues and future values', () => {
	it('resolves unknown customisation values to safe native defaults', () => {
		expect(
			effectiveCustomisation({
				colour: 'new-colour',
				profileBackground: '../../private'
			})
		).toEqual({ colour: 'cobalt', background: 'none' });
		expect(
			effectiveCustomisation({
				colour: 'rose',
				profileBackground: 'craft-knitting'
			})
		).toEqual({ colour: 'rose', background: 'craft-knitting' });
		expect(effectiveCustomisation(null)).toEqual({
			colour: 'cobalt',
			background: 'none'
		});
	});
	it('offers actual supported ISO currencies with readable names', () => {
		expect(
			currencyOptions.find((value) => value.code === 'GBP')?.label
		).toMatch(/^GBP - British Pound/);
		expect(
			currencyOptions.find((value) => value.code === 'JPY')?.label
		).toMatch(/^JPY - Japanese Yen/);
		expect(new Set(currencyOptions.map((value) => value.code)).size).toBe(
			currencyOptions.length
		);
		expect(
			currencyOptions.every((value) => /^[A-Z]{3}$/.test(value.code))
		).toBe(true);
	});
	it('uses controlled moderation labels and readable unknown fallbacks', () => {
		expect(moderationLabel('reason', 'suspected_ai_generated')).toBe(
			'Suspected AI-generated content'
		);
		expect(moderationLabel('effect', 'visibilityTakedown')).toBe(
			'Content removed from CraftSky'
		);
		expect(moderationLabel('action', 'restore')).toBe('Access restored');
		expect(moderationLabel('reason', 'illegal_content')).toBe('Other illegal content');
		expect(moderationLabel('reason', 'fraud_or_scam')).toBe('Fraud or scam');
		expect(moderationLabel('reason', 'self_harm')).toBe('Self-harm policy violation');
		expect(moderationLabel('reason', 'future-policy')).toBe('Policy decision');
		expect(moderationLabel('effect', null)).toBe('Policy action');
	});
});

describe('native follower growth presentation', () => {
	it('pads integer axes, marks observations and leaves missing days disconnected', () => {
		const chart = followerChart(
			[
				{ date: '2026-11-01', count: 8 },
				{ date: '2026-11-02', count: null },
				{ date: '2026-11-03', count: 10 }
			],
			'7d',
			402
		)!;
		expect(chart.lower).toBe(7);
		expect(chart.upper).toBe(11);
		expect(chart.ticks.map((value) => value.count)).toEqual([7, 8, 9, 10, 11]);
		expect(chart.dots).toHaveLength(2);
		expect(chart.paths).toHaveLength(2);
		expect(chart.paths.every((value) => !value.includes(' L'))).toBe(true);
		expect(chart.dates.map((value) => value.date)).toEqual([
			'2026-11-01',
			'2026-11-02',
			'2026-11-03'
		]);
	});
	it('uses each month’s latest known count for the year view', () => {
		const chart = followerChart(
			[
				{ date: '2026-09-01', count: 10 },
				{ date: '2026-09-20', count: 12 },
				{ date: '2026-09-30', count: null },
				{ date: '2026-10-01', count: null },
				{ date: '2026-11-01', count: 15 }
			],
			'1y'
		)!;
		expect(chart.dots).toHaveLength(2);
		expect(chart.paths).toHaveLength(2);
		expect(chart.dates.map((value) => value.date)).toEqual([
			'2026-09-01',
			'2026-10-01',
			'2026-11-01'
		]);
	});
	it('handles no observations and a zero constant without invalid coordinates', () => {
		expect(
			followerChart([{ date: '2026-11-01', count: null }])
		).toBeUndefined();
		const chart = followerChart([
			{ date: '2026-11-01', count: 0 },
			{ date: '2026-11-02', count: 0 }
		])!;
		expect(chart.lower).toBe(0);
		expect(chart.upper).toBe(1);
		expect(chart.paths.join('')).not.toMatch(/NaN|Infinity/);
	});
});
