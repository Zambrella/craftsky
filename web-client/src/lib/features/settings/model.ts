import { FieldValidationError } from '#lib/field-validation.ts';
import { APIError, MutationOutcomeUnknownError } from '#lib/api/client.ts';
import type { JsonObject } from '#lib/api/types.ts';
import languageCatalogue from './languages.json';
import currencyScales from './currencies.json';
import { requireRecordText } from './text-limits.ts';
export const languageOptions = Object.entries(languageCatalogue)
	.map(([tag, name]) => ({ tag, name }))
	.sort((a, b) => a.name.localeCompare(b.name));

export type Session = { id?: string; did: string; handle: string };
export const title = (value: string) =>
	value
		.replace(/([a-z])([A-Z])/g, '$1 $2')
		.replace(/[-_]/g, ' ')
		.replace(/^./, (c) => c.toUpperCase());
export const message = (error: unknown): string =>
	error instanceof APIError || error instanceof Error
		? error.message
		: 'The request could not be completed. Please try again.';
export const dateTime = (value: unknown): string =>
	typeof value === 'string' && Number.isFinite(Date.parse(value))
		? new Intl.DateTimeFormat(undefined, {
				dateStyle: 'medium',
				timeStyle: value.length > 10 ? 'short' : undefined
			}).format(new Date(value))
		: 'Not available';
export const record = (value: unknown): Record<string, unknown> =>
	value !== null && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: {};
export const text = (value: unknown): string =>
	typeof value === 'string' ? value : '';
export const openValue = (value: unknown): string =>
	typeof value === 'string' ? value : text(record(value).value);
export const https = (value: string): boolean => {
	try {
		const u = new URL(value);
		return (
			u.protocol === 'https:' && !!u.hostname && !u.username && !u.password
		);
	} catch {
		return false;
	}
};
export const validDid = (value: string): boolean =>
	/^did:[a-z]+:[A-Za-z0-9._:%-]+$/.test(value);

/** Invalidates late completions when a view or account changes. */
export function accountScope() {
	let generation = 0;
	return {
		token: () => generation,
		current: (token: number) => generation === token,
		invalidate: () => {
			generation++;
		}
	};
}
/** A potentially accepted PDS write must retain the original payload and operation key. */
export function pendingWrite(error: unknown): boolean {
	return error instanceof MutationOutcomeUnknownError;
}

export const colours: Record<string, string> = {
	cobalt: '#1535d6',
	orchid: '#b615d6',
	rose: '#d61535',
	amber: '#766200',
	lime: '#23770f',
	teal: '#007663',
	ink: '#161210'
};
export const backgrounds = [
	'none',
	'bayerdark',
	'cubedark',
	'dotcrossdark',
	'scallopdark',
	'skewdark',
	'x2',
	'craft-sewing',
	'craft-knitting',
	'craft-crochet',
	'craft-quilting',
	'craft-embroidery'
];
export const businessTypes = [
	'dyer',
	'fiber-producer',
	'fiber-processor',
	'yarn-shop',
	'fabric-shop',
	'craft-supply-shop',
	'pattern-designer',
	'finished-goods-maker',
	'tool-maker',
	'teacher',
	'craft-studio',
	'repair-service',
	'technical-editor',
	'photographer',
	'publisher',
	'other-craft-business'
];
export const offerings = [
	'yarn',
	'fiber',
	'fabric',
	'patterns',
	'kits',
	'notions',
	'tools',
	'finished-goods',
	'custom-work',
	'repairs',
	'classes',
	'studio-hire',
	'wholesale',
	'digital-products',
	'technical-editing',
	'photography-services',
	'fiber-processing'
];
export const actions = [
	'shop',
	'browse-patterns',
	'request-custom-order',
	'book-class',
	'book-appointment',
	'view-event-calendar',
	'email',
	'visit-website',
	'wholesale-enquiries'
];
export const roles = [
	'organizer',
	'instructor',
	'vendor',
	'exhibitor',
	'speaker',
	'demonstrator'
];

export interface ParsedFollowing {
	entries: { username: string }[];
	ignored: number;
	duplicates: number;
}
export function parseFollowing(source: string, json = false): ParsedFollowing {
	if (new TextEncoder().encode(source).byteLength > 20 * 1024 * 1024)
		throw new Error('Choose a file smaller than 20 MB.');
	let values: unknown[];
	if (!json) values = source.split(/\r?\n/);
	else {
		let parsed: unknown;
		try {
			parsed = JSON.parse(source);
		} catch {
			throw new Error(
				'This file is not valid JSON. Choose Instagram’s following.json file.'
			);
		}
		const rows = record(parsed).relationships_following;
		if (!Array.isArray(rows))
			throw new Error(
				'Choose Instagram’s following.json file containing relationships_following.'
			);
		values = rows.flatMap((row) => {
			const r = record(row),
				data = r.string_list_data;
			if (!Array.isArray(data)) return [null];
			const legacy = data.filter((d) => Object.hasOwn(record(d), 'value'));
			if (legacy.length) return legacy.map((d) => record(d).value);
			const name = text(r.title).replace(/^@/, '').toLowerCase();
			const href = data.length === 1 ? text(record(data[0]).href) : '';
			return [
				href ===
					`https://www.instagram.com/_u/${text(r.title).replace(/^@/, '')}` &&
				/^[a-z0-9._]{1,30}$/i.test(name)
					? name
					: null
			];
		});
	}
	const seen = new Set<string>();
	let ignored = 0,
		duplicates = 0;
	for (const value of values) {
		const username =
			typeof value === 'string'
				? value.trim().replace(/^@/, '').toLowerCase()
				: '';
		if (!/^[a-z0-9._]{1,30}$/.test(username)) {
			ignored++;
			continue;
		}
		if (seen.has(username)) {
			duplicates++;
			continue;
		}
		seen.add(username);
		if (seen.size > 10000)
			throw new Error('Import at most 10,000 Instagram usernames at a time.');
	}
	return {
		entries: [...seen].map((username) => ({ username })),
		ignored,
		duplicates
	};
}

export interface GrowthPoint {
	date: string;
	count: number | null;
}
export function growthSegments(points: GrowthPoint[]): string[] {
	const values = points.flatMap((p) => (p.count === null ? [] : [p.count]));
	if (!values.length) return [];
	const min = Math.min(...values),
		span = Math.max(1, Math.max(...values) - min);
	const paths: string[] = [];
	let path = '';
	points.forEach((p, i) => {
		if (p.count === null) {
			if (path) paths.push(path);
			path = '';
			return;
		}
		path += `${path ? ' L' : 'M'}${20 + (i * 560) / Math.max(1, points.length - 1)},${180 - ((p.count - min) * 150) / span}`;
	});
	if (path) paths.push(path);
	return paths;
}

export function languagePreferences(primary: string, source: string) {
	primary = primary.toLowerCase();
	const values = [
		...new Set(
			source
				.toLowerCase()
				.split(/[,\s]+/)
				.filter(Boolean)
		)
	];
	const tags = [primary, ...values];
	if (tags.some((tag) => !Object.hasOwn(languageCatalogue, tag)))
		throw new Error(
			'Choose supported languages, such as English, French or Portuguese.'
		);
	return { primaryLanguage: primary, contentLanguages: values };
}

export interface BusinessImage {
	cid: string;
	mime: string;
	size: number;
	alt: string;
	thumb?: string;
	aspectRatio?: { width: number; height: number };
}
export function imageBody(image: BusinessImage): JsonObject {
	requireRecordText(image.alt, 'Alternative text', 1000, 10000);
	if (
		!['image/jpeg', 'image/png', 'image/webp'].includes(image.mime) ||
		image.size < 0 ||
		image.size > 2000000
	)
		throw new Error('Use a valid image smaller than 2 MB.');
	return {
		image: {
			$type: 'blob',
			ref: { $link: image.cid },
			mimeType: image.mime,
			size: image.size
		},
		alt: image.alt,
		...(image.aspectRatio ? { aspectRatio: image.aspectRatio } : {})
	};
}
export interface Product {
	title: string;
	uri: string;
	image?: BusinessImage;
	amount: string;
	currency: string;
}
export function productBody(products: Product[]): JsonObject[] {
	if (products.length > 4)
		throw new Error('You can feature up to four products.');
	const destinations = new Set<string>();
	return products.map((p) => {
		requireRecordText(p.title, 'Product title', 150, 1500);
		if (!p.title.trim())
			throw new FieldValidationError(
				'Title',
				'Each product needs a title of up to 150 characters.'
			);
		if (!https(p.uri))
			throw new FieldValidationError(
				'Destination link',
				'Each product needs a secure HTTPS destination.'
			);
		if (destinations.has(p.uri))
			throw new FieldValidationError(
				'Destination link',
				'Each product needs a different destination.'
			);
		destinations.add(p.uri);
		if (!p.image)
			throw new FieldValidationError(
				'Product image',
				'Add an image to each product.'
			);
		let amount = p.amount
			.trim()
			.replace(/(\.\d*?)0+$/, '$1')
			.replace(/\.$/, '');
		const currency = p.currency.trim().toUpperCase();
		if (!!amount !== !!currency)
			throw new FieldValidationError(
				'Price',
				'Enter both the price and currency, or leave both blank.'
			);
		if (
			amount &&
			(!/^(0|[1-9][0-9]{0,11})(\.\d{1,4})?$/.test(amount) ||
				!Object.hasOwn(currencyScales, currency) ||
				(amount.split('.')[1]?.length || 0) >
					(currencyScales[currency as keyof typeof currencyScales] ?? 0))
		)
			throw new FieldValidationError(
				'Price',
				'Use a valid decimal price and three-letter currency code.'
			);
		return {
			title: p.title,
			uri: p.uri,
			image: imageBody(p.image),
			...(amount ? { price: { amount, currency } } : {})
		};
	});
}

/** Converts a wall-clock form value in an IANA timezone, rejecting DST gaps. */
export function zonedTimestamp(value: string, zone: string): string {
	const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value);
	if (!match) throw new Error('Choose a valid date and time.');
	const values = match.slice(1).map(Number),
		target = Date.UTC(
			values[0],
			values[1] - 1,
			values[2],
			values[3],
			values[4]
		);
	const check = new Date(target);
	if (
		check.getUTCFullYear() !== values[0] ||
		check.getUTCMonth() !== values[1] - 1 ||
		check.getUTCDate() !== values[2] ||
		values[3] > 23 ||
		values[4] > 59
	)
		throw new Error('Choose a valid date and time.');
	let candidate = target;
	const format = new Intl.DateTimeFormat('en-CA', {
		timeZone: zone,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23'
	});
	const represented = (at: number) => {
		const parts = Object.fromEntries(
			format.formatToParts(at).map((p) => [p.type, p.value])
		);
		return Date.UTC(
			+parts.year,
			+parts.month - 1,
			+parts.day,
			+parts.hour,
			+parts.minute
		);
	};
	for (let i = 0; i < 4; i++) candidate += target - represented(candidate);
	if (represented(candidate) !== target)
		throw new Error(
			'This time does not exist in the selected time zone. Choose another time.'
		);
	return new Date(candidate).toISOString();
}
export function wallTime(value: unknown, zone: string): string {
	if (typeof value !== 'string' || !Number.isFinite(Date.parse(value)))
		return '';
	const p = Object.fromEntries(
		new Intl.DateTimeFormat('en-CA', {
			timeZone: zone,
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			hourCycle: 'h23'
		})
			.formatToParts(new Date(value))
			.map((p) => [p.type, p.value])
	);
	return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`;
}
