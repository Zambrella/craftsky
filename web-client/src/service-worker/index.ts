/// <reference lib="webworker" />
import { initializeApp } from '@firebase/app';
import { getMessaging, onBackgroundMessage } from '@firebase/messaging/sw';
import * as publicEnv from '$app/env/public';
import { pushConfig } from '../lib/features/push/model.ts';

// No fetch handler: authenticated pages, messages and API responses are never cached.
const config = pushConfig(publicEnv);
if (config) {
	const app = initializeApp(config, 'craftsky-web-push');
	const messaging = getMessaging(app);
	onBackgroundMessage(messaging, () => {
		/* FCM displays the server notification once and handles its verified HTTPS click URL. */
	});
}
