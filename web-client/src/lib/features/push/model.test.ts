import { describe, it, expect } from 'vitest';
import {
	pushConfig,
	bindings,
	saveBinding,
	removeBinding,
	pushTarget,
	pushURL,
	targetFromURL,
	accountForTarget
} from './model.ts';
const a = '00000000-0000-4000-8000-000000000001',
	b = '00000000-0000-4000-8000-000000000002',
	subA = '00000000-0000-4000-8000-000000000003',
	subB = '00000000-0000-4000-8000-000000000004';
function storage() {
	const values = new Map<string, string>();
	return {
		getItem: (key: string) => values.get(key) || null,
		setItem: (key: string, value: string) => {
			values.set(key, value);
		}
	};
}
describe('browser push configuration', () => {
	it('is disabled unless every required public configuration value exists', () => {
		expect(pushConfig({})).toBeNull();
		const env = {
			PUBLIC_FIREBASE_API_KEY: 'public-key',
			PUBLIC_FIREBASE_APP_ID: 'app',
			PUBLIC_FIREBASE_PROJECT_ID: 'project',
			PUBLIC_FIREBASE_MESSAGING_SENDER_ID: 'sender',
			PUBLIC_FIREBASE_VAPID_KEY: 'public-vapid'
		};
		expect(pushConfig(env)?.vapidKey).toBe('public-vapid');
		expect(pushConfig({ ...env, PUBLIC_FIREBASE_VAPID_KEY: '' })).toBeNull();
	});
});
describe('private multiaccount push routing', () => {
	it('routes to the enabled owner using an opaque subscription ID, independent of the active account', () => {
		const data = storage();
		saveBinding(data, { accountId: a, subscriptionId: subA, enabled: true });
		saveBinding(data, { accountId: b, subscriptionId: subB, enabled: true });
		const target = pushTarget({
			accountSubscriptionId: subA,
			type: 'like',
			recipientDid: 'must-not-be-used',
			text: 'must-not-be-stored'
		});
		expect(target).toEqual({ subscriptionId: subA });
		expect(accountForTarget(bindings(data), target!)).toBe(a);
		expect(JSON.stringify(bindings(data))).not.toContain('must-not');
		removeBinding(data, a);
		expect(accountForTarget(bindings(data), target!)).toBeNull();
	});
	it('rejects malformed payloads, unbound subscriptions and unverified routing strings', () => {
		expect(
			pushTarget({
				accountSubscriptionId: 'https://evil.example',
				type: 'like'
			})
		).toBeNull();
		expect(
			pushTarget({
				accountSubscriptionId: subA,
				type: 'chat',
				openReference: 'javascript:evil'
			})
		).toBeNull();
		expect(
			targetFromURL(
				new URL(
					'https://app.craftsky.social/notifications?pushSubscription=invalid'
				)
			)
		).toBeNull();
		expect(
			accountForTarget(
				[{ accountId: a, subscriptionId: subA, enabled: false }],
				{ subscriptionId: subA }
			)
		).toBeNull();
	});
	it('only produces relative application destinations with validated opaque references', () => {
		const target = { subscriptionId: subA, reference: subB };
		expect(
			targetFromURL(new URL(pushURL(target), 'https://app.craftsky.social'))
		).toEqual(target);
		expect(pushURL(target)).toMatch(/^\/messages\?/);
	});
	it('rotates registration bindings without retaining stale subscription routes', () => {
		const data = storage();
		saveBinding(data, { accountId: a, subscriptionId: subA, enabled: true });
		saveBinding(data, { accountId: a, subscriptionId: subB, enabled: true });
		expect(bindings(data)).toEqual([
			{ accountId: a, subscriptionId: subB, enabled: true }
		]);
	});
});
