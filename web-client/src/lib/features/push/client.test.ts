import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';
import type { CraftskyAPI } from '#lib/api/client.ts';
const sdk = vi.hoisted(() => ({
	getToken: vi.fn(async () => 'test-browser-fcm-registration'),
	deleteToken: vi.fn(async () => true),
	onMessage: vi.fn(() => () => {})
}));
vi.mock('$app/env/public', () => ({
	PUBLIC_FIREBASE_API_KEY: 'public-key',
	PUBLIC_FIREBASE_APP_ID: 'app',
	PUBLIC_FIREBASE_PROJECT_ID: 'project',
	PUBLIC_FIREBASE_MESSAGING_SENDER_ID: 'sender',
	PUBLIC_FIREBASE_VAPID_KEY: 'public-vapid',
	PUBLIC_FIREBASE_AUTH_DOMAIN: ''
}));
vi.mock('@firebase/messaging', () => ({
	...sdk,
	isSupported: async () => true,
	getMessaging: () => ({})
}));
vi.mock('@firebase/app', () => ({
	getApps: () => [],
	initializeApp: () => ({})
}));
const a = '00000000-0000-4000-8000-000000000001',
	b = '00000000-0000-4000-8000-000000000002',
	subA = '00000000-0000-4000-8000-000000000003',
	subB = '00000000-0000-4000-8000-000000000004';
const dispatched = vi.fn(),
	permission = vi.fn(async () => 'granted');
beforeEach(() => {
	vi.resetModules();
	vi.clearAllMocks();
	const values = new Map<string, string>();
	vi.stubGlobal('localStorage', {
		getItem: (key: string) => values.get(key) || null,
		setItem: (key: string, value: string) => values.set(key, value)
	});
	vi.stubGlobal('Notification', {
		permission: 'granted',
		requestPermission: permission
	});
	vi.stubGlobal('window', {
		Notification,
		isSecureContext: true,
		dispatchEvent: dispatched
	});
	vi.stubGlobal('navigator', {
		serviceWorker: {
			register: async () => ({ scope: '/' }),
			ready: Promise.resolve({})
		}
	});
});
afterEach(() => vi.unstubAllGlobals());
describe('browser push lifecycle', () => {
	it('renews a registration without asking for permission or producing an event loop', async () => {
		const { registerPush } = await import('./client.ts');
		const registerDevice = vi.fn(async () => ({ accountSubscriptionId: subA }));
		const api = { notifications: { registerDevice } } as unknown as CraftskyAPI;
		await registerPush(api, a, true);
		permission.mockClear();
		await registerPush(api, a);
		expect(registerDevice).toHaveBeenCalledTimes(2);
		expect(registerDevice).toHaveBeenCalledWith(
			'web',
			'test-browser-fcm-registration'
		);
		expect(permission).not.toHaveBeenCalled();
		expect(dispatched).toHaveBeenCalledTimes(1);
	});
	it('requests permission only from the explicit opt-in operation', async () => {
		const { registerPush } = await import('./client.ts');
		const api = {
			notifications: {
				registerDevice: async () => ({ accountSubscriptionId: subA })
			}
		} as unknown as CraftskyAPI;
		await registerPush(api, a, true);
		expect(permission).toHaveBeenCalledTimes(1);
	});
	it('unregisters only the requested account and retains a token needed by another account', async () => {
		const { registerPush, disablePush } = await import('./client.ts');
		let count = 0;
		const unregisterDevice = vi.fn(async () => {});
		const api = {
			notifications: {
				registerDevice: async () => ({
					accountSubscriptionId: ++count === 1 ? subA : subB
				}),
				unregisterDevice
			}
		} as unknown as CraftskyAPI;
		await registerPush(api, a, true);
		await registerPush(api, b, true);
		await disablePush(api, a);
		expect(unregisterDevice).toHaveBeenCalledWith(subA);
		expect(sdk.deleteToken).not.toHaveBeenCalled();
		await disablePush(api, b);
		expect(sdk.deleteToken).toHaveBeenCalledTimes(1);
	});
	it('keeps a renewal queued behind disable from re-enabling notifications', async () => {
		const { registerPush, disablePush } = await import('./client.ts');
		const registerDevice = vi.fn(async () => ({ accountSubscriptionId: subA }));
		let finish!: () => void;
		const removal = new Promise<void>((resolve) => (finish = resolve));
		const api = {
			notifications: { registerDevice, unregisterDevice: () => removal }
		} as unknown as CraftskyAPI;
		await registerPush(api, a, true);
		const disabled = disablePush(api, a);
		const renewal = registerPush(api, a);
		finish();
		await Promise.all([disabled, renewal]);
		expect(registerDevice).toHaveBeenCalledTimes(1);
		expect(sdk.deleteToken).toHaveBeenCalledTimes(1);
	});
	it('clears a stale local binding when the server already removed the subscription', async () => {
		const { registerPush, disablePush } = await import('./client.ts');
		const { bindings } = await import('./model.ts');
		const { APIError } = await import('#lib/api/transport.ts');
		const api = {
			notifications: {
				registerDevice: async () => ({ accountSubscriptionId: subA }),
				unregisterDevice: async () => {
					throw new APIError(404, 'subscription_not_found', 'fixture');
				}
			}
		} as unknown as CraftskyAPI;
		await registerPush(api, a, true);
		await disablePush(api, a);
		expect(bindings(localStorage)).toEqual([]);
		expect(sdk.deleteToken).toHaveBeenCalledTimes(1);
	});
});

it('treats denied notification permission as expected without starting registration', async () => {
	const { registerPush } = await import('./client.ts');
	const { expectedFailure } = await import('../diagnostics/privacy');
	permission.mockResolvedValueOnce('denied');
	const error = await registerPush(
		{} as CraftskyAPI,
		'private-account',
		true
	).catch((error) => error);
	expect(expectedFailure(error)).toBe(true);
	expect(error.message).toContain('Notifications are blocked');
});
