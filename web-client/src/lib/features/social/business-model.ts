import type { JsonObject } from '#lib/api/types.ts';
import { safeLink } from './model';
import type { BusinessEvent } from '#lib/api/types.ts';

export function eventDisplay(
	event: Pick<BusinessEvent, 'startsAt' | 'endsAt' | 'isAllDay'>,
	locale?: string
) {
	const start = new Date(event.startsAt),
		end = new Date(Date.parse(event.endsAt) - (event.isAllDay ? 86400000 : 0));
	if (!Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime()))
		return { date: 'Date unavailable', time: '' };
	const date = (value: Date) =>
		value.toLocaleDateString(locale, {
			day: 'numeric',
			month: 'short',
			year: 'numeric'
		});
	const time = (value: Date) =>
		value.toLocaleTimeString(locale, { hour: 'numeric', minute: '2-digit' });
	return {
		date:
			event.isAllDay && date(start) !== date(end)
				? `${date(start)} – ${date(end)}`
				: date(start),
		time: event.isAllDay ? 'All day' : `${time(start)} – ${time(end)}`
	};
}
export function countryName(value: unknown): string {
	if (typeof value !== 'string') return '';
	try {
		return (
			new Intl.DisplayNames(undefined, { type: 'region' }).of(value) || value
		);
	} catch {
		return value;
	}
}
export function money(value: JsonObject | undefined): string {
	if (
		!value ||
		typeof value.amount !== 'string' ||
		typeof value.currency !== 'string' ||
		!/^\d+(?:\.\d+)?$/.test(value.amount)
	)
		return '';
	try {
		return new Intl.NumberFormat(undefined, {
			style: 'currency',
			currency: value.currency
		}).format(Number(value.amount));
	} catch {
		return `${value.amount} ${value.currency}`;
	}
}
export const businessLabels = (value: string) =>
	value
		.split('#')
		.pop()!
		.replace(/([a-z])([A-Z])/g, '$1 $2')
		.replaceAll('-', ' ')
		.replace(/^./, (letter) => letter.toUpperCase());
const actions: Record<string, string> = {
	shop: 'Shop',
	'browse-patterns': 'Browse patterns',
	'request-custom-order': 'Request a custom order',
	'book-class': 'Book a class',
	'book-appointment': 'Book an appointment',
	'view-event-calendar': 'View event calendar',
	email: 'Email',
	'visit-website': 'Visit website',
	'wholesale-enquiries': 'Wholesale enquiries'
};
export function businessAction(
	value: JsonObject | undefined
): { href: string; label: string } | undefined {
	if (
		!value ||
		typeof value.type !== 'string' ||
		!Object.hasOwn(actions, value.type) ||
		typeof value.destination !== 'string'
	)
		return;
	let href = safeLink(value.destination);
	if (value.type === 'email') {
		const raw = value.destination;
		if (
			!raw.startsWith('mailto:') ||
			raw.length > 327 ||
			/[\s%?#,;\\]/.test(raw)
		)
			return;
		const email = raw.slice(7);
		if (
			!/^[!$&'*+\-/=^_`{|}~A-Za-z0-9]+(?:\.[!$&'*+\-/=^_`{|}~A-Za-z0-9]+)*@(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/.test(
				email
			)
		)
			return;
		href = raw;
	}
	return href ? { href, label: actions[value.type] } : undefined;
}
