import { describe, expect, it, vi } from 'vitest';
import { defaultProject, options } from './model';
import { craftChoice, primaryCrafts } from './project-options';
import { browseFilters, filterOptions } from '../social/filter-model';

vi.mock('./project-taxonomy.json', async (load) => {
	const { default: entries } = await load<{
		default: { family: string; phase?: number }[];
	}>();
	return {
		default: entries.map((entry) =>
			entry.family === 'craft' && entry.phase === 2
				? { ...entry, offered: false }
				: entry
		)
	};
});

describe('Phase 1 project craft choices', () => {
	it('offers only the original crafts while retaining labels for known records', () => {
		expect(primaryCrafts).toHaveLength(5);
		expect(options('craft')).toHaveLength(5);
		expect(craftChoice('weaving')?.label).toBe('Weaving');
	});

	it('skips a disabled preferred craft for new projects', () => {
		expect(defaultProject(['weaving', 'crochet'])).toMatchObject({
			common: { craftType: 'social.craftsky.feed.defs#crochet' }
		});
		expect(defaultProject(['spinning']).common).not.toHaveProperty('craftType');
	});

	it('omits Phase 2 filter choices from old links', () => {
		expect(filterOptions('feltingMethod', 'felting')).toEqual([]);
		expect(filterOptions('projectType', 'punch-needle')).toEqual([]);
		expect(filterOptions('loomType', 'weaving')).toEqual([]);
		expect(filterOptions('knittingMethod', 'knitting').length).toBeGreaterThan(
			0
		);
		expect(browseFilters(new URLSearchParams('feltingMethod=needle'))).toEqual(
			{}
		);
		expect(
			browseFilters(
				new URLSearchParams(
					'craft=weaving&loomType=rigidHeddle&settUnit=ends-per-inch'
				)
			)
		).toEqual({});
	});
});
