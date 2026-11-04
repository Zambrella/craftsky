import { describe, expect, it } from 'vitest';
import { searchDestination, recentDestination } from './search-model';
import type { RecentSearch } from '#lib/api/types.ts';

describe('native search intent', () => {
	it('submits trimmed queries to Posts and routes leading handles/tags to the matching tab', () => {
		expect(searchDestination('  @alice  ')).toBe(
			'/app/search?q=%40alice&tab=profiles'
		);
		expect(searchDestination('#knitting')).toBe(
			'/app/search?q=%23knitting&tab=hashtags'
		);
		expect(searchDestination('linen dress', 'popular')).toBe(
			'/app/search?q=linen+dress&tab=posts&sort=popular'
		);
		expect(searchDestination('   ')).toBeUndefined();
	});
	it('opens query history with native default intent while retaining explicit project searches', () => {
		expect(
			recentDestination({
				id: 'query-1',
				displayLabel: '@alice',
				type: 'query',
				payload: { q: '@alice' }
			} as RecentSearch)
		).toBe('/app/search?q=%40alice&tab=profiles');
		expect(
			recentDestination({
				id: 'project-1',
				displayLabel: 'linen',
				type: 'project',
				payload: { q: 'linen' }
			} as RecentSearch)
		).toBe('/app/search?q=linen&tab=projects');
	});
});
