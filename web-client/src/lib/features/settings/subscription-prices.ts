import type { WebSubscriptionPrice } from '#lib/api/types.ts';
import scales from './currencies.json';

/** Stripe amounts use currency minor units, including its ISK/UGX exception. */
export function monthlyPrice(
	price: WebSubscriptionPrice | undefined,
	locale = 'en-GB'
) {
	if (!price || !Number.isSafeInteger(price.amount) || price.amount <= 0)
		return null;
	const currency = price.currency.toUpperCase();
	const digits =
		currency === 'ISK' || currency === 'UGX'
			? 2
			: (scales as Record<string, number>)[currency];
	if (digits === undefined) return null;
	return `${new Intl.NumberFormat(locale, { style: 'currency', currency }).format(price.amount / 10 ** digits)} / month`;
}
