import { describe, expect, it } from 'vitest';
import type { AppNotification, Profile } from '#lib/api/types.ts';
import { notificationCopy, profileTabs, joinedAge } from './presentation';

describe('native social presentation policy', () => {
	it('uses the native joined-age thresholds and handles invalid/future dates', () => {
		const now = Date.parse('2026-11-02T12:00:00Z');
		expect(joinedAge('2026-11-02T14:00:00Z', now)).toBe('today');
		expect(joinedAge('2026-10-31T12:00:00Z', now)).toBe('2d');
		expect(joinedAge('2026-09-01T12:00:00Z', now)).toBe('2mo');
		expect(joinedAge('2025-11-02T12:00:00Z', now)).toBe('1y');
		expect(joinedAge('invalid', now)).toBe('');
	});
	it('keeps normal/blocked tabs and orders business tabs by actual availability', () => {
		const profile = {
			did: 'did:plc:a',
			accountType: 'business',
			business: { products: [] }
		} as unknown as Profile;
		expect(profileTabs(profile, false)).toEqual([
			'about',
			'projects',
			'posts',
			'comments',
			'reposts'
		]);
		expect(profileTabs(profile, true)).toEqual([
			'products',
			'events',
			'about',
			'projects',
			'posts',
			'comments',
			'reposts'
		]);
		expect(profileTabs({ ...profile, hasUpcomingEvents: true }, false)[0]).toBe(
			'events'
		);
		expect(profileTabs({ ...profile, blocking: true }, false)).toEqual([
			'projects',
			'posts',
			'comments',
			'reposts'
		]);
	});
	it('uses human notification copy and never turns unknown wire values into visible prose', () => {
		expect(
			notificationCopy({
				type: 'like',
				preview: { role: 'comment' }
			} as AppNotification).action
		).toBe('liked your comment');
		expect(
			notificationCopy({ type: 'PRIVATE_ENUM_CANARY' } as AppNotification)
				.action
		).toBe('New activity');
		expect(
			notificationCopy({
				type: 'instagram_match',
				actor: { did: 'did:plc:a' }
			} as AppNotification).canFollow
		).toBe(true);
	});
	it('does not resurface protected post preview text or blocked-actor follow actions', () => {
		const item = {
			type: 'follow',
			actor: { blocking: true },
			subjectPost: { availability: 'blocked' },
			preview: { text: 'PRIVATE_PREVIEW_CANARY' }
		} as unknown as AppNotification;
		expect(notificationCopy(item)).toMatchObject({
			canFollow: false,
			preview: ''
		});
	});
	it('keeps parent context distinct from a new reply and quotes the original content role', () => {
		const reply = notificationCopy({
			type: 'reply',
			subjectPost: {
				uri: 'at://did:plc:a/social.craftsky.feed.post/parent',
				author: { did: 'did:plc:a' },
				text: 'The parent project',
				project: { common: { title: 'Dinosaur blanket' } }
			},
			preview: { text: 'The new comment', role: 'comment' }
		} as AppNotification);
		expect(reply.context).toBe('↳ Your post: Dinosaur blanket');
		expect(reply.previewText).toBe('The new comment');
		expect(
			notificationCopy({
				type: 'quote',
				preview: { text: 'New quote', role: 'post' },
				quotedPreview: { text: 'Original reply', role: 'reply' }
			} as AppNotification).action
		).toBe('quoted your reply');
	});
	it('honours explicit unavailable wire fields across all preview and follow surfaces', () => {
		for (const unavailable of [
			{ actor: { available: false } },
			{ contentAvailable: false }
		]) {
			const result = notificationCopy({
				type: 'quote',
				...unavailable,
				preview: { text: 'PRIVATE_PREVIEW_CANARY' },
				quotedPreview: { text: 'PRIVATE_QUOTE_CANARY' }
			} as AppNotification);
			expect(result).toMatchObject({
				available: false,
				action: 'Activity unavailable',
				canFollow: false,
				preview: '',
				previewText: '',
				context: ''
			});
			expect(JSON.stringify(result)).not.toContain('PRIVATE_');
		}
	});
});
