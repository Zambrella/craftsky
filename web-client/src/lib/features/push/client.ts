import { InputError, UserFacingError } from '#lib/user-facing-error.ts';
import type { CraftskyAPI } from '#lib/api/client.ts';
import { APIError } from '#lib/api/transport.ts';
import {
	bindings,
	pushConfig,
	saveBinding,
	removeBinding,
	uuid,
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
		if (!config)
			throw new UserFacingError('Browser notifications are not configured.');
		if (
			!('serviceWorker' in navigator) ||
			!('Notification' in window) ||
			!window.isSecureContext
		)
			throw new InputError(
				'This browser does not support notifications. On iPhone or iPad, add CraftSky to your Home Screen first.'
			);
		const sdk = await import('@firebase/messaging');
		if (!(await sdk.isSupported()))
			throw new InputError(
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
let registrationQueue = Promise.resolve();
function withRegistrationLock(operation: () => Promise<void>): Promise<void> {
	if (navigator.locks)
		return navigator.locks.request('craftsky-browser-push', operation);
	const next = registrationQueue.then(operation);
	registrationQueue = next.catch(() => {});
	return next;
}
export async function registerPush(
	api: CraftskyAPI,
	accountId: string,
	requestPermission = false
) {
	if (requestPermission) {
		if (!('Notification' in window))
			throw new InputError('This browser does not support notifications.');
		const result = await Notification.requestPermission();
		if (result !== 'granted')
			throw new InputError(
				result === 'denied'
					? 'Notifications are blocked. Allow notifications in your browser settings to enable them.'
					: 'Notification permission was not granted.'
			);
	}
	return withRegistrationLock(async () => {
		const previous = bindings(localStorage).find(
			(row) => row.accountId === accountId
		);
		if (!requestPermission && !previous?.enabled) return;
		if (!('Notification' in window) || Notification.permission !== 'granted') {
			if (!requestPermission) return;
			throw new InputError('Notification permission was not granted.');
		}
		const { messaging: instance, registration, config } = await messaging();
		const { getToken } = await import('@firebase/messaging');
		const token = await getToken(instance, {
			vapidKey: config.vapidKey,
			serviceWorkerRegistration: registration
		});
		if (!token)
			throw new UserFacingError(
				'The browser could not create a notification registration.'
			);
		const { accountSubscriptionId } = await api.notifications.registerDevice(
			'web',
			token
		);
		if (previous?.enabled && previous.subscriptionId === accountSubscriptionId)
			return;
		try {
			saveBinding(localStorage, {
				accountId,
				subscriptionId: accountSubscriptionId,
				enabled: true
			});
		} catch (error) {
			// A registration that we cannot route locally is unusable.
			if (uuid(accountSubscriptionId))
				await api.notifications
					.unregisterDevice(accountSubscriptionId)
					.catch(() => {});
			throw error;
		}
		window.dispatchEvent(new Event('craftsky-push-change'));
	});
}
export async function disablePush(api: CraftskyAPI, accountId: string) {
	return withRegistrationLock(async () => {
		const binding = bindings(localStorage).find(
			(row) => row.accountId === accountId
		);
		if (!binding) return;
		try {
			await api.notifications.unregisterDevice(binding.subscriptionId);
		} catch (error) {
			if (
				!(error instanceof APIError) ||
				error.status !== 404 ||
				error.code !== 'subscription_not_found'
			)
				throw error;
		}
		await removeLocalPush(accountId);
	});
}
/** Local cleanup after successful logout, including when server revocation was already authoritative. */
export async function forgetPush(accountId: string) {
	return withRegistrationLock(async () => {
		if (bindings(localStorage).some((row) => row.accountId === accountId))
			await removeLocalPush(accountId);
	});
}
async function removeLocalPush(accountId: string) {
	removeBinding(localStorage, accountId);
	window.dispatchEvent(new Event('craftsky-push-change'));
	// Other signed-in accounts may share this browser token. Remove it only when none remain.
	if (!bindings(localStorage).some((row) => row.enabled) && configuration()) {
		try {
			const { messaging: instance } = await messaging();
			const { deleteToken } = await import('@firebase/messaging');
			await deleteToken(instance);
		} catch {}
	}
}
export async function listenPush(receive: (data: unknown) => void) {
	const { messaging: instance } = await messaging();
	const { onMessage } = await import('@firebase/messaging');
	return onMessage(instance, (payload) => receive(payload.data));
}
