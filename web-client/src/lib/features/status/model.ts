export interface StatusMessage {
	revision: string;
	minVersion: string;
	maxVersion?: string;
	title: string;
	description: string;
	imageUrl?: string;
	cta?: { text: string; url?: string; destination?: string };
}
export interface ServiceStatus {
	schemaVersion: 2;
	revision: string;
	maintenance?: {
		title: string;
		description: string;
		estimatedRecoveryAt?: string;
	};
	minimumVersions?: { web?: string };
	message?: StatusMessage;
}
const object = (value: unknown): value is Record<string, unknown> =>
	!!value && typeof value === 'object' && !Array.isArray(value);
const revision = (value: unknown) =>
	typeof value === 'string' && /^[A-Za-z0-9_-]{1,64}$/.test(value);
const text = (value: unknown, max: number) =>
	typeof value === 'string' &&
	!!value.trim() &&
	[...value].length <= max &&
	!/[\uD800-\uDFFF]/u.test(value);
const number = '(?:0|[1-9][0-9]{0,14})';
const identifier = '(?:' + number + '|[0-9]*[A-Za-z-][0-9A-Za-z-]*)';
const syntax = new RegExp(
	'^(' +
		number +
		')\\.(' +
		number +
		')\\.(' +
		number +
		')(?:-(' +
		identifier +
		'(?:\\.' +
		identifier +
		')*))?(?:\\+[0-9A-Za-z-]+(?:\\.[0-9A-Za-z-]+)*)?$'
);
export function version(value: unknown): { core: number[]; pre: string[] } {
	const match = typeof value === 'string' ? syntax.exec(value) : null;
	if (!match) throw new Error('Invalid version');
	return {
		core: match.slice(1, 4).map(Number),
		pre: match[4]?.split('.') || []
	};
}
export function compare(a: string, b: string) {
	const aa = version(a),
		bb = version(b);
	for (let i = 0; i < 3; i++)
		if (aa.core[i] !== bb.core[i]) return aa.core[i] < bb.core[i] ? -1 : 1;
	if (!aa.pre.length && !bb.pre.length) return 0;
	if (!aa.pre.length) return 1;
	if (!bb.pre.length) return -1;
	for (let i = 0; i < Math.max(aa.pre.length, bb.pre.length); i++) {
		const x = aa.pre[i],
			y = bb.pre[i];
		if (x === undefined) return -1;
		if (y === undefined) return 1;
		if (x === y) continue;
		const nx = /^\d+$/.test(x),
			ny = /^\d+$/.test(y);
		if (nx && ny) return Number(x) < Number(y) ? -1 : 1;
		if (nx !== ny) return nx ? -1 : 1;
		return x < y ? -1 : 1;
	}
	return 0;
}
function validTimestamp(value: unknown) {
	if (typeof value !== 'string') return false;
	const match =
		/^(\d{4})-(\d\d)-(\d\d)T(\d\d):(\d\d):(\d\d)(?:\.\d{1,6})?(Z|[+-]\d\d:\d\d)$/.exec(
			value
		);
	if (!match) return false;
	const [year, month, day, hour, minute, second] = match
		.slice(1, 7)
		.map(Number);
	const date = new Date(0);
	date.setUTCFullYear(year, month - 1, day);
	const zone = match[7];
	return (
		year >= 1 &&
		date.getUTCFullYear() === year &&
		date.getUTCMonth() === month - 1 &&
		date.getUTCDate() === day &&
		hour < 24 &&
		minute < 60 &&
		second < 60 &&
		(zone === 'Z' ||
			(Number(zone.slice(1, 3)) < 24 && Number(zone.slice(4)) < 60)) &&
		Number.isFinite(Date.parse(value))
	);
}
export function https(value: unknown): value is string {
	if (
		typeof value !== 'string' ||
		/[^\x21-\x7e]|\\|%(?![0-9a-f]{2})/i.test(value)
	)
		return false;
	try {
		const url = new URL(value);
		return (
			url.protocol === 'https:' &&
			!!url.hostname &&
			!url.username &&
			!url.password
		);
	} catch {
		return false;
	}
}
export const destinations: Record<string, string> = {
	feed: '/feed',
	projects: '/feed',
	search: '/search',
	notifications: '/notifications',
	profile: '/profile',
	settings: '/settings'
};
export function parseStatus(value: unknown): ServiceStatus {
	if (!object(value) || value.schemaVersion !== 2 || !revision(value.revision))
		throw new Error('Invalid status document');
	if (value.maintenance !== undefined) {
		const item = value.maintenance;
		if (
			!object(item) ||
			!text(item.title, 160) ||
			!text(item.description, 2400)
		)
			throw new Error('Invalid maintenance');
		if (
			item.estimatedRecoveryAt !== undefined &&
			!validTimestamp(item.estimatedRecoveryAt)
		)
			throw new Error('Invalid estimate');
	}
	if (value.minimumVersions !== undefined) {
		if (!object(value.minimumVersions)) throw new Error('Invalid minima');
		for (const platform of ['web', 'ios', 'android'])
			if (value.minimumVersions[platform] !== undefined)
				version(value.minimumVersions[platform]);
	}
	if (value.message !== undefined) {
		const item = value.message;
		if (
			!object(item) ||
			!revision(item.revision) ||
			!text(item.title, 160) ||
			!text(item.description, 2400)
		)
			throw new Error('Invalid message');
		version(item.minVersion);
		if (item.maxVersion !== undefined) {
			version(item.maxVersion);
			if (compare(item.minVersion as string, item.maxVersion as string) > 0)
				throw new Error('Invalid range');
		}
		if (item.imageUrl !== undefined && !https(item.imageUrl))
			throw new Error('Invalid image');
		if (item.cta !== undefined) {
			const cta = item.cta;
			if (
				!object(cta) ||
				!text(cta.text, 160) ||
				'url' in cta === 'destination' in cta ||
				('url' in cta
					? !https(cta.url)
					: typeof cta.destination !== 'string' ||
						!Object.hasOwn(destinations, cta.destination))
			)
				throw new Error('Invalid action');
		}
	}
	return value as unknown as ServiceStatus;
}
export async function readStatus(response: Response): Promise<ServiceStatus> {
	if (
		response.status !== 200 ||
		response.headers.get('content-type')?.split(';')[0].trim().toLowerCase() !==
			'application/json' ||
		!response.body
	)
		throw new Error('Invalid response');
	const reader = response.body.getReader();
	let size = 0;
	const chunks: Uint8Array[] = [];
	try {
		for (;;) {
			const next = await reader.read();
			if (next.done) break;
			size += next.value.byteLength;
			if (size > 16384) throw new Error('Oversized status');
			chunks.push(next.value);
		}
	} finally {
		await reader.cancel().catch(() => {});
		reader.releaseLock();
	}
	const bytes = new Uint8Array(size);
	let at = 0;
	for (const chunk of chunks) {
		bytes.set(chunk, at);
		at += chunk.length;
	}
	return parseStatus(
		JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes))
	);
}
