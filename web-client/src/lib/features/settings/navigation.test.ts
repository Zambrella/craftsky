import { describe, expect, it } from 'vitest';
import { moderationSubject, storeManagementURL } from './navigation.ts';

describe('reviewed settings destinations', () => {
	it('offers only official existing-subscription management pages', () => {
		expect(storeManagementURL('app_store')).toBe(
			'https://apps.apple.com/account/subscriptions'
		);
		expect(storeManagementURL('play_store')).toBe(
			'https://play.google.com/store/account/subscriptions'
		);
		expect(
			storeManagementURL('https://private.example/checkout')
		).toBeUndefined();
	});
	it('shows the affected content reference and links only a matching typed owner', () => {
		const snapshot = {
			type: 'post',
			did: 'did:plc:maker',
			uri: 'at://did:plc:maker/social.craftsky.feed.post/example'
		};
		expect(moderationSubject(snapshot)).toEqual({
			reference: snapshot.uri,
			href: '/posts/did%3Aplc%3Amaker/example',
			label: 'View post'
		});
		expect(moderationSubject({ ...snapshot, did: 'did:plc:other' })).toEqual({
			reference: snapshot.uri
		});
		expect(
			moderationSubject({ ...snapshot, uri: 'javascript:alert(1)' }).href
		).toBeUndefined();
		expect(
			moderationSubject({
				type: 'account',
				did: 'did:plc:maker',
				submittedHandle: 'maker.example'
			})
		).toEqual({
			reference: 'maker.example',
			href: '/profiles/did%3Aplc%3Amaker',
			label: 'View profile'
		});
	});
});
