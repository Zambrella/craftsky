import { describe, expect, it } from 'vitest';
import {
	searchDestination,
	recentDestination,
	recentAuthor
} from './search-model';
import type { RecentSearch } from '#lib/api/types.ts';

describe('native search intent', () => {
	it('submits trimmed queries to Posts and routes leading handles/tags to the matching tab', () => {
		expect(searchDestination('  @alice  ')).toBe(
			'/search?q=%40alice&tab=profiles'
		);
		expect(searchDestination('#knitting')).toBe(
			'/search?q=%23knitting&tab=hashtags'
		);
		expect(searchDestination('linen dress', 'popular')).toBe(
			'/search?q=linen+dress&tab=posts&sort=popular'
		);
		expect(searchDestination('   ')).toBeUndefined();
	});
	it('retains hydrated profile decorations with safe defaults for old history', () => {
		const recent: RecentSearch = {
			id: 'profile',
			type: 'profile',
			displayLabel: 'Bob',
			payload: {
				did: 'did:plc:bob',
				handle: 'bob.test',
				accountType: 'business',
				customisation: { colour: 'orchid', profileBackground: 'animated-flow' }
			}
		};
		expect(recentAuthor(recent)).toMatchObject({
			accountType: 'business',
			customisation: recent.payload.customisation
		});
		expect(
			recentAuthor({
				...recent,
				payload: { did: 'did:plc:bob', handle: 'bob.test' }
			})
		).toMatchObject({ accountType: undefined, customisation: undefined });
		expect(
			recentAuthor({
				...recent,
				payload: {
					did: 'did:plc:bob',
					handle: 'bob.test',
					accountType: 'future',
					customisation: []
				}
			})
		).toMatchObject({ accountType: undefined, customisation: undefined });
	});
	it('opens query history with native default intent while retaining explicit project searches', () => {
		expect(
			recentDestination({
				id: 'query-1',
				displayLabel: '@alice',
				type: 'query',
				payload: { q: '@alice' }
			} as RecentSearch)
		).toBe('/search?q=%40alice&tab=profiles');
		expect(
			recentDestination({
				id: 'project-1',
				displayLabel: 'linen',
				type: 'project',
				payload: { q: 'linen' }
			} as RecentSearch)
		).toBe('/search?q=linen&tab=projects');
	});
});
