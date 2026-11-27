/// <reference lib="webworker" />
import { initializeApp } from '@firebase/app';
import { getMessaging, onBackgroundMessage } from '@firebase/messaging/sw';
import * as publicEnv from '$app/env/public';
import { pushConfig } from '../lib/features/push/model.ts';
import {
	backgroundNotification,
	notificationClickRoute
} from '../lib/features/push/worker.ts';

// No fetch handler: authenticated pages, messages and API responses are never cached.
const config = pushConfig(publicEnv);
if (config) {
	const worker = self as unknown as ServiceWorkerGlobalScope;
	// The worker owns push delivery only. Open tabs do not need to wait for an
	// application cache transition before receiving the updated handlers.
	worker.addEventListener('install', (event) => {
		event.waitUntil(worker.skipWaiting());
	});
	worker.addEventListener('notificationclick', (event) => {
		const destination = notificationClickRoute(event.notification.data);
		if (!destination) return;
		event.notification.close();
		event.waitUntil(worker.clients.openWindow(destination));
	});
	const app = initializeApp(config, 'craftsky-web-push');
	const messaging = getMessaging(app);
	onBackgroundMessage(messaging, (payload) => {
		const notification = backgroundNotification(payload.data);
		if (!notification) return;
		return worker.registration.showNotification(
			notification.title,
			notification.options
		);
	});
}
