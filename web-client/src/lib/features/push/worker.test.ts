import { afterEach, expect, it, vi } from 'vitest';

const firebase = vi.hoisted(() => ({
	onBackgroundMessage: vi.fn()
}));
vi.mock('@firebase/messaging/sw', () => ({
	getMessaging: () => ({}),
	onBackgroundMessage: firebase.onBackgroundMessage
}));
vi.mock('@firebase/app', () => ({ initializeApp: () => ({}) }));
vi.mock('$app/env/public', () => ({
	PUBLIC_FIREBASE_API_KEY: 'public-key',
	PUBLIC_FIREBASE_APP_ID: 'app',
	PUBLIC_FIREBASE_PROJECT_ID: 'project',
	PUBLIC_FIREBASE_MESSAGING_SENDER_ID: 'sender',
	PUBLIC_FIREBASE_VAPID_KEY: 'public-vapid',
	PUBLIC_FIREBASE_AUTH_DOMAIN: ''
}));
afterEach(() => vi.unstubAllGlobals());

it('displays web pushes once and opens their validated route regardless of existing tabs', async () => {
	const addEventListener = vi.fn();
	const openWindow = vi.fn(async () => null);
	const showNotification = vi
		.fn<typeof ServiceWorkerRegistration.prototype.showNotification>()
		.mockResolvedValue(undefined);
	vi.stubGlobal('self', {
		addEventListener,
		clients: { openWindow },
		registration: { showNotification }
	});
	await import('../../../service-worker/index.ts');
	const [, show] = firebase.onBackgroundMessage.mock.calls[0];
	const [, click] = addEventListener.mock.calls.find(
		([name]) => name === 'notificationclick'
	)!;
	const accountSubscriptionId = '00000000-0000-4000-8000-000000000001';
	const openReference = '00000000-0000-4000-8000-000000000002';
	await show({
		data: {
			type: 'chat',
			accountSubscriptionId,
			openReference,
			webTitle: 'CraftSky',
			webBody: 'New message'
		}
	});
	expect(showNotification).toHaveBeenCalledExactlyOnceWith(
		'CraftSky',
		expect.objectContaining({
			body: 'New message',
			data: { type: 'craftsky-push', accountSubscriptionId, openReference }
		})
	);
	const close = vi.fn();
	const waitUntil = vi.fn();
	click({
		notification: { data: showNotification.mock.calls[0][1]?.data, close },
		waitUntil
	});
	expect(close).toHaveBeenCalledTimes(1);
	expect(openWindow).toHaveBeenCalledExactlyOnceWith(
		`/messages?pushSubscription=${accountSubscriptionId}&pushReference=${openReference}`
	);
	expect(waitUntil).toHaveBeenCalledWith(openWindow.mock.results[0].value);

	await show({
		data: {
			type: 'chat',
			accountSubscriptionId,
			webTitle: 'CraftSky',
			webBody: 'New message'
		}
	});
	click({
		notification: {
			data: { type: 'craftsky-push', accountSubscriptionId: '//invalid.test' },
			close
		},
		waitUntil
	});
	expect(showNotification).toHaveBeenCalledTimes(1);
	expect(openWindow).toHaveBeenCalledTimes(1);
});
