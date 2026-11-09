import { describe, expect, it } from 'vitest';
import {
	backgroundLabels,
	notificationCategories,
	notificationLabels,
	visibleSettingsGroups
} from './presentation.ts';
import { backgrounds } from './model.ts';

describe('native settings navigation policy', () => {
	it('keeps the native preferences, connections, discovery, business and general ordering', () => {
		const groups = visibleSettingsGroups(false, true);
		expect(groups.map((group) => group.title)).toEqual([
			'Preferences',
			'Connections',
			'Discovery',
			'Business',
			'General'
		]);
		expect(
			groups
				.find((group) => group.title === 'Business')
				?.items.map(([slug]) => slug)
		).toEqual(['events', 'products']);
		expect(
			groups
				.find((group) => group.title === 'General')
				?.items.map(([slug]) => slug)
		).toEqual(['safety', 'moderation', 'account', 'about', 'feedback']);
	});
	it('does not advertise business management without authoritative Business access', () => {
		expect(
			visibleSettingsGroups(false, false).some(
				(group) => group.title === 'Business'
			)
		).toBe(false);
	});
	it('preserves recovery, moderation and relationship reversal for restricted accounts', () => {
		const slugs = visibleSettingsGroups(true, true).flatMap((group) =>
			group.items.map(([slug]) => slug)
		);
		expect(slugs).toEqual([
			'muted',
			'blocked',
			'safety',
			'moderation',
			'account',
			'about',
			'feedback'
		]);
	});
});

describe('native preference presentation contracts', () => {
	it('uses native notification order and labels independently of server object insertion order', () => {
		expect(notificationCategories).toEqual([
			'like',
			'follow',
			'reply',
			'mention',
			'quote',
			'repost',
			'instagramMatch',
			'moderation',
			'everythingElse'
		]);
		expect(notificationLabels.reply).toBe('Comments & replies');
		for (const category of notificationCategories)
			expect(notificationLabels[category]).toBeTruthy();
	});
	it('labels every persisted background choice without exposing asset identifiers', () => {
		for (const value of backgrounds)
			expect(backgroundLabels[value]).toBeTruthy();
		expect(backgroundLabels.dotcrossdark).toBe('Cross stitch');
		expect(backgroundLabels.skewdark).toBe('Diagonal weave');
	});
});
