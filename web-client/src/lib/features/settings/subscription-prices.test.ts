import { describe, expect, it } from 'vitest';
import { monthlyPrice } from './subscription-prices';
import type { WebSubscriptionPrice } from '#lib/api/types.ts';
const price = (amount: number, currency: string): WebSubscriptionPrice => ({
	tier: 'plus',
	amount,
	currency,
	taxBehavior: 'inclusive'
});
describe('web monthly catalog price', () => {
	it('formats the server amount and currency without hard-coded plan prices', () => {
		expect(monthlyPrice(price(249, 'gbp'))).toBe('£2.49 / month');
		expect(monthlyPrice(price(379, 'gbp'))).toBe('£3.79 / month');
		expect(monthlyPrice(price(500, 'jpy'))).toContain('500 / month');
		expect(monthlyPrice(price(2999, 'bhd'))).toContain('2.999 / month');
		expect(monthlyPrice(price(50000, 'isk'))).toContain('500 / month');
	});
	it('never invents a price for absent or invalid catalog data', () => {
		for (const value of [
			undefined,
			price(-1, 'gbp'),
			price(0, 'gbp'),
			price(2.49, 'gbp'),
			price(Infinity, 'gbp'),
			price(249, 'invalid')
		])
			expect(monthlyPrice(value)).toBeNull();
	});
});
