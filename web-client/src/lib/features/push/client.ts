import type { CraftskyAPI } from '#lib/api/client.ts';
import {
	bindings,
	pushConfig,
	saveBinding,
	removeBinding,
	type PushConfig
} from './model.ts';
import * as publicEnv from '$app/env/public';

export const configuration = () => pushConfig(publicEnv);
let setup:
	| Promise<{
			messaging: import('@firebase/messaging').Messaging;
			registration: ServiceWorkerRegistration;
			config: PushConfig;
	  }>
	| undefined;
async function messaging() {
	if (setup) return setup;
	setup = (async () => {
		const config = configuration();
		if (!config) throw new Error('Browser notifications are not configured.');
		if (
			!('serviceWorker' in navigator) ||
			!('Notification' in window) ||
			!window.isSecureContext
		)
			throw new Error(
				'This browser does not support notifications. On iPhone or iPad, add CraftSky to your Home Screen first.'
			);
		const sdk = await import('@firebase/messaging');
		if (!(await sdk.isSupported()))
			throw new Error(
				'This browser does not support notifications. On iPhone or iPad, add CraftSky to your Home Screen first.'
			);
		const { initializeApp, getApps } = await import('@firebase/app');
		const app =
			getApps().find((app) => app.name === 'craftsky-web-push') ||
			initializeApp(config, 'craftsky-web-push');
		const registration = await navigator.serviceWorker.register(
			'/service-worker.js',
			{ scope: '/', type: 'module' }
		);
		await navigator.serviceWorker.ready;
		return { messaging: sdk.getMessaging(app), registration, config };
	})().catch((error) => {
		setup = undefined;
		throw error;
	});
	return setup;
}
export async function registerPush(
	api: CraftskyAPI,
	accountId: string,
	requestPermission = false
) {
	if (requestPermission) {
		if (!('Notification' in window))
			throw new Error('This browser does not support notifications.');
		const result = await Notification.requestPermission();
		if (result !== 'granted')
			throw new Error(
				result === 'denied'
					? 'Notifications are blocked. Allow notifications in your browser settings to enable them.'
					: 'Notification permission was not granted.'
			);
	}
	if (Notification.permission !== 'granted') return;
	const { messaging: instance, registration, config } = await messaging();
	const { getToken } = await import('@firebase/messaging');
	const token = await getToken(instance, {
		vapidKey: config.vapidKey,
		serviceWorkerRegistration: registration
	});
	if (!token)
		throw new Error(
			'The browser could not create a notification registration.'
		);
	const previous = bindings(localStorage).find(
		(row) => row.accountId === accountId
	);
	const response = await api.notifications.registerDevice('web', token);
	saveBinding(localStorage, {
		accountId,
		subscriptionId: response.accountSubscriptionId,
		enabled: true
	});
	if (
		!previous?.enabled ||
		previous.subscriptionId !== response.accountSubscriptionId
	)
		window.dispatchEvent(new Event('craftsky-push-change'));
}
export async function disablePush(api: CraftskyAPI, accountId: string) {
	const binding = bindings(localStorage).find(
		(row) => row.accountId === accountId
	);
	if (binding) await api.notifications.unregisterDevice(binding.subscriptionId);
	await forgetPush(accountId);
}
/** Local cleanup after successful logout, including when server revocation was already authoritative. */
export async function forgetPush(accountId: string) {
	removeBinding(localStorage, accountId);
	// Other signed-in accounts may share this browser token. Remove it only when none remain.
	if (!bindings(localStorage).some((row) => row.enabled) && configuration()) {
		try {
			const { messaging: instance } = await messaging();
			const { deleteToken } = await import('@firebase/messaging');
			await deleteToken(instance);
		} catch {}
	}
	window.dispatchEvent(new Event('craftsky-push-change'));
}
export async function listenPush(receive: (data: unknown) => void) {
	const { messaging: instance } = await messaging();
	const { onMessage } = await import('@firebase/messaging');
	return onMessage(instance, (payload) => receive(payload.data));
}
