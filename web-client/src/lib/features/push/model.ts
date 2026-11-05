export type PushConfig = {
	apiKey: string;
	appId: string;
	projectId: string;
	messagingSenderId: string;
	vapidKey: string;
	authDomain?: string;
};
const configFields = {
	apiKey: 'PUBLIC_FIREBASE_API_KEY',
	appId: 'PUBLIC_FIREBASE_APP_ID',
	projectId: 'PUBLIC_FIREBASE_PROJECT_ID',
	messagingSenderId: 'PUBLIC_FIREBASE_MESSAGING_SENDER_ID',
	vapidKey: 'PUBLIC_FIREBASE_VAPID_KEY'
} as const;
export function pushConfig(env: Record<string, unknown>): PushConfig | null {
	const result: Record<string, string> = {};
	for (const [key, name] of Object.entries(configFields)) {
		const value = env[name];
		if (typeof value !== 'string' || !value.trim()) return null;
		result[key] = value.trim();
	}
	if (typeof env.PUBLIC_FIREBASE_AUTH_DOMAIN === 'string')
		result.authDomain = env.PUBLIC_FIREBASE_AUTH_DOMAIN;
	return result as PushConfig;
}
export const uuid = (value: unknown): value is string =>
	typeof value === 'string' &&
	/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(
		value
	);
export type PushBinding = {
	accountId: string;
	subscriptionId: string;
	enabled: boolean;
};
const key = 'craftsky.push.bindings';
export type BindingStorage = {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
};
/** Only opaque browser-account and routing IDs are stored. No credentials or notification contents. */
export function bindings(storage: BindingStorage): PushBinding[] {
	try {
		const rows: unknown = JSON.parse(storage.getItem(key) || '[]');
		return Array.isArray(rows)
			? rows
					.filter(
						(row): row is PushBinding =>
							!!row &&
							typeof row === 'object' &&
							uuid(row.accountId) &&
							uuid(row.subscriptionId) &&
							typeof row.enabled === 'boolean'
					)
					.slice(0, 8)
					.map((row) => ({
						accountId: row.accountId,
						subscriptionId: row.subscriptionId,
						enabled: row.enabled
					}))
			: [];
	} catch {
		return [];
	}
}
export function saveBinding(storage: BindingStorage, binding: PushBinding) {
	if (!uuid(binding.accountId) || !uuid(binding.subscriptionId))
		throw new Error('CraftSky returned an invalid notification registration.');
	storage.setItem(
		key,
		JSON.stringify(
			[
				...bindings(storage).filter(
					(row) => row.accountId !== binding.accountId
				),
				{
					accountId: binding.accountId,
					subscriptionId: binding.subscriptionId,
					enabled: binding.enabled
				}
			].slice(-8)
		)
	);
}
export function removeBinding(storage: BindingStorage, accountId: string) {
	storage.setItem(
		key,
		JSON.stringify(
			bindings(storage).filter((row) => row.accountId !== accountId)
		)
	);
}
export type PushTarget = { subscriptionId: string; reference?: string };
export function pushTarget(value: unknown): PushTarget | null {
	if (!value || typeof value !== 'object') return null;
	const data = value as Record<string, unknown>;
	if (!uuid(data.accountSubscriptionId)) return null;
	if (data.type === 'chat') {
		if (!uuid(data.openReference)) return null;
		return {
			subscriptionId: data.accountSubscriptionId,
			reference: data.openReference
		};
	}
	return { subscriptionId: data.accountSubscriptionId };
}
export function targetFromURL(url: URL): PushTarget | null {
	const subscriptionId = url.searchParams.get('pushSubscription');
	const reference = url.searchParams.get('pushReference');
	if (!uuid(subscriptionId) || (reference !== null && !uuid(reference)))
		return null;
	return { subscriptionId, ...(reference ? { reference } : {}) };
}
export function pushURL(target: PushTarget): string {
	const params = new URLSearchParams({
		pushSubscription: target.subscriptionId
	});
	if (target.reference) params.set('pushReference', target.reference);
	return `${target.reference ? '/messages' : '/notifications'}?${params}`;
}
export function accountForTarget(
	rows: PushBinding[],
	target: PushTarget
): string | null {
	return (
		rows.find(
			(row) => row.enabled && row.subscriptionId === target.subscriptionId
		)?.accountId || null
	);
}
