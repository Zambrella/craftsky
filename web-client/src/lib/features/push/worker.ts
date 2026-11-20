import { pushTarget, pushURL } from './model.ts';

/** The sender's web-only data message is displayed by our service worker. */
export function backgroundNotification(data: Record<string, string> = {}) {
	const target = pushTarget(data);
	if (!target || !data.webTitle || !data.webBody) return null;
	return {
		title: data.webTitle,
		options: {
			body: data.webBody,
			icon: '/app-icon.png',
			data: {
				type: 'craftsky-push',
				accountSubscriptionId: target.subscriptionId,
				...(target.reference ? { openReference: target.reference } : {})
			}
		}
	};
}

export function notificationClickRoute(value: unknown): string | null {
	if (!value || typeof value !== 'object') return null;
	const data = value as Record<string, unknown>;
	if (data.type !== 'craftsky-push') return null;
	const target = pushTarget({
		...data,
		type: data.openReference === undefined ? 'activity' : 'chat'
	});
	return target && pushURL(target);
}
