import { describe, it, expect } from 'vitest';
import {
	browseFilters,
	filterOptions,
	filterValueLabel,
	projectItemFilterChoices,
	selectedProjectItems
} from './filter-model';
import { craftFeeds, craftKey } from './craft-feeds';
import { pinSlot, quoteCanRender, visible, facetParts, texture } from './model';
import type { Post } from '#lib/api/client.ts';
import { publicReadPath } from '#lib/api/public-path.ts';
import { craftFilterFamilies } from './craft-detail-filters';
import { craftChoice } from '../composer/project-options';
import { matchingSelectOptions } from '#lib/select-options.ts';
describe('project browse controls and wire values', () => {
	it('keeps canonical item and method filters while dropping retired URL filters', () => {
		const params = new URLSearchParams({
			craft: 'crochet',
			crochetMethod: 'tunisianCrochet',
			hookSizeMm: '4.0mm',
			designTag: 'floral'
		});
		expect(browseFilters(params)).toEqual({
			crochetMethod: ['tunisianCrochet']
		});
		expect(filterOptions('color', 'sewing')).toEqual([]);
		expect(filterOptions('designTag', 'crochet')).toEqual([]);
	});
	it('maps difficulty and scopes named subtypes to their exact craft', () => {
		expect(
			filterOptions('patternDifficulty', 'knitting').map(({ id }) => id)
		).toContain('social.craftsky.feed.defs#intermediate');
		const sewing = filterOptions('projectSubtype', 'sewing');
		expect(sewing.length).toBeGreaterThan(0);
		expect(
			sewing.every(({ craft }) => craft === 'social.craftsky.feed.defs#sewing')
		).toBe(true);
		const garment = 'social.craftsky.project.defs#garment';
		expect(
			projectItemFilterChoices('sewing', []).some(
				({ parentId }) => parentId === garment
			)
		).toBe(true);
	});
	it('keeps context for alias matches and keeps the full branch for a type match', () => {
		const options = projectItemFilterChoices('knitting', []);
		const garment = 'social.craftsky.project.defs#garment';
		const branch = options.filter(
			({ id, parentId }) => id === garment || parentId === garment
		);
		expect(branch.length).toBeGreaterThan(1);
		expect(matchingSelectOptions(options, 'Garment')).toEqual(branch);
		expect(
			matchingSelectOptions(options, 'pullover').map(({ label }) => label)
		).toEqual(['Garment', 'Jumper']);
		expect(options.at(-1)?.label).toBe('Other');
	});
	it('sends canonical combined choices on project browse, including guest reads', () => {
		const garment = 'social.craftsky.project.defs#garment';
		const jumper = 'social.craftsky.project.knitting.defs#pullover';
		const params = new URLSearchParams([
			['projectItem', 'garment'],
			['projectItem', jumper],
			['projectItem', garment],
			['projectItem', 'not a catalogue option']
		]);
		expect(browseFilters(params, 'knitting')).toEqual({
			projectItem: [garment, jumper]
		});
		expect(publicReadPath('projects', params)).toBe('projects?' + params);
		expect(() => publicReadPath('search/projects', params)).toThrow(TypeError);
	});
	it('omits empty and unsupported parameters and preserves repeated filters', () => {
		expect(
			browseFilters(
				new URLSearchParams(
					'craft=sewing&status=&projectType=garment&projectType=accessory&projectType=garment&selfDrafted=false&private=ignored'
				)
			)
		).toEqual({ projectType: ['garment', 'accessory'] });
		expect(browseFilters(new URLSearchParams('selfDrafted=true'))).toEqual({
			selfDrafted: true
		});
	});
	it('keeps named historical filters visible and ignores free-form item names as filters', () => {
		const beanie = 'social.craftsky.project.knitting.defs#beanie';
		expect(
			filterOptions('projectSubtype', 'knitting').map(({ id }) => id)
		).not.toContain(beanie);
		expect(
			browseFilters(
				new URLSearchParams(
					'craft=knitting&projectSubtype=beanie&projectSubtype=Lampshade'
				)
			)
		).toEqual({ projectSubtype: ['beanie'] });
		expect(filterValueLabel('projectSubtype', 'knitting', 'beanie')).toBe(
			'Beanie'
		);
		const values = selectedProjectItems('knitting', {
			projectType: ['accessory'],
			projectSubtype: ['beanie']
		});
		expect(values).toEqual([beanie]);
		expect(
			projectItemFilterChoices('knitting', values).find(
				({ id }) => id === beanie
			)?.selectionLabel
		).toBe('Hat · Beanie');
		expect(
			browseFilters(
				new URLSearchParams('craft=sewing&projectSubtype=' + beanie)
			)
		).toEqual({});
	});
	it('keeps preferred craft feeds first and lists active craft filters', () => {
		const ordered = craftFeeds(['punch-needle', 'weaving']).map((entry) =>
			craftKey(entry.id)
		);
		expect(ordered.slice(0, 2)).toEqual(['punch-needle', 'weaving']);
		expect(new Set(ordered).size).toBe(11);
		const expected = {
			knitting: ['knittingMethod'],
			crochet: ['crochetMethod'],
			sewing: [],
			embroidery: ['embroideryMethod', 'embroideryStyle'],
			quilting: [],
			spinning: [],
			dyeing: [],
			'punch-needle': [],
			macrame: [],
			weaving: ['loomType'],
			felting: ['feltingMethod']
		};
		for (const [craft, keys] of Object.entries(expected)) {
			expect(
				craftFilterFamilies
					.filter(([key]) => filterOptions(key, craft).length)
					.map(([key]) => key)
			).toEqual(craftChoice(craft)?.offered === false ? [] : keys);
		}
		expect(
			filterOptions('knittingMethod', 'knitting').map(({ labels }) => labels.en)
		).toEqual(['Hand', 'Machine', 'Loom']);
		expect(
			filterOptions('crochetMethod', 'crochet').map(({ labels }) => labels.en)
		).toEqual(['Standard', 'Tunisian']);
		expect(
			filterOptions('feltingMethod', 'felting').map(({ labels }) => labels.en)
		).toEqual(['Needle', 'Wet']);
	});
	it('keeps Embroidery style selectable independently of method', () => {
		expect(
			filterOptions('embroideryStyle', 'embroidery').map(({ id }) =>
				id.split('#').at(-1)
			)
		).toEqual(['crossStitch', 'needlepoint']);
		expect(
			browseFilters(
				new URLSearchParams('craft=embroidery&embroideryStyle=sashiko')
			)
		).toEqual({ embroideryStyle: ['sashiko'] });
	});
	it('keeps craft selections on the guest browse request only', () => {
		const params = new URLSearchParams([
			['craft', 'crochet'],
			['crochetMethod', 'tunisianCrochet'],
			['hookSizeMm', '4.0mm']
		]);
		expect(browseFilters(params)).toEqual({
			crochetMethod: ['tunisianCrochet']
		});
		params.delete('craft');
		expect(publicReadPath('projects', params)).toBe(
			'projects?' + params.toString()
		);
		expect(() => publicReadPath('search/projects', params)).toThrow(TypeError);
	});
	it('offers types for ten applicable crafts and no item picker for Spinning', () => {
		for (const { id } of craftFeeds()) {
			const types = filterOptions('projectType', id);
			if (craftKey(id) === 'spinning') expect(types).toEqual([]);
			else expect(types.at(-1)?.id).toBe('social.craftsky.project.defs#other');
		}
		expect(
			filterOptions('projectType', 'punch-needle').map(
				({ labels }) => labels.en
			)
		).toEqual([
			'Rug or floor mat',
			'Wall hanging',
			'Hoop art',
			'Cushion or seat pad',
			'Coaster',
			'Table mat or runner',
			'Other'
		]);
		expect(filterOptions('projectSubtype', 'punch-needle')).toEqual([]);
	});
});
describe('post presentation and actions', () => {
	it('matches Flutter pin eligibility for standard and project roots', () => {
		expect(pinSlot({})).toBe('standard');
		expect(pinSlot({ project: {} as Post['project'] })).toBe('project');
		expect(
			pinSlot({
				reply: {
					root: { uri: 'root', cid: 'cid' },
					parent: { uri: 'root', cid: 'cid' }
				}
			})
		).toBeUndefined();
		expect(
			pinSlot({
				project: {} as Post['project'],
				quote: { uri: 'target', cid: 'cid' }
			})
		).toBeUndefined();
	});
	it('hides warned, blocked and stale quote payloads', () => {
		const quoted = {
			uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
			author: { did: 'did:plc:alice', handle: 'alice.test' },
			rkey: 'root'
		} as Post;
		expect(visible(quoted)).toBe(true);
		expect(visible({ ...quoted, availability: 'blocked' })).toBe(false);
		expect(
			quoteCanRender({
				quoteView: { state: 'visible', revealable: false, post: quoted }
			} as unknown as Post)
		).toBe(true);
		for (const state of ['blocked', 'muted', 'unavailable'])
			expect(
				quoteCanRender({
					quoteView: { state, revealable: true, post: quoted }
				} as unknown as Post)
			).toBe(false);
		expect(
			quoteCanRender({
				quoteView: {
					state: 'visible',
					revealable: true,
					post: { ...quoted, moderation: { warningKind: 'sensitive' } }
				}
			} as unknown as Post)
		).toBe(false);
	});
	it('links exact hashtag feeds and safely allowlists craft texture assets', () => {
		expect(
			facetParts('#knitting', [
				{
					index: { byteStart: 0, byteEnd: 9 },
					features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'knitting' }]
				}
			])[0].href
		).toBe('/search/tags?tag=knitting');
		for (const craft of [
			'knitting',
			'crochet',
			'sewing',
			'embroidery',
			'quilting',
			'spinning',
			'dyeing',
			'punch-needle',
			'macrame',
			'weaving',
			'felting'
		])
			expect(
				texture({ colour: 'cobalt', profileBackground: 'craft-' + craft })
			).toBe(`url('/profile_backgrounds/craft-${craft}.svg')`);
		expect(
			texture({ colour: 'cobalt', profileBackground: 'craft-../../private' })
		).toBeUndefined();
	});
});
