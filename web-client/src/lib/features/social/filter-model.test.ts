import { describe, it, expect } from 'vitest';
import { browseFilters, filterOptions } from './filter-model';
import { pinSlot, quoteCanRender, visible, facetParts, texture } from './model';
import type { Post } from '#lib/api/client.ts';
describe('project browse controls and wire values', () => {
	it('maps difficulty and scopes subtypes to their exact craft', () => {
		expect(
			filterOptions('patternDifficulty', 'knitting').map((value) => value.id)
		).toContain('social.craftsky.feed.defs#intermediate');
		const sewing = filterOptions('projectSubtype', 'sewing');
		expect(sewing.length).toBeGreaterThan(0);
		expect(
			sewing.every(
				(value) => value.craft === 'social.craftsky.feed.defs#sewing'
			)
		).toBe(true);
		expect(filterOptions('yarnWeight', 'sewing')).toEqual([]);
		expect(filterOptions('piecingTechnique', 'knitting')).toEqual([]);
		expect(
			filterOptions(
				'projectSubtype',
				'sewing',
				'social.craftsky.project.defs#garment'
			).every((value) =>
				value.parents.includes('social.craftsky.project.defs#garment')
			)
		).toBe(true);
	});
	it('unions subtype choices across every selected parent project type', () => {
		const parents = [
			'social.craftsky.project.defs#garment',
			'social.craftsky.project.defs#accessory'
		];
		const expected = new Set(
			parents.flatMap((parent) =>
				filterOptions('projectSubtype', 'knitting', parent).map(
					(option) => option.id
				)
			)
		);
		expect(expected.size).toBeGreaterThan(0);
		expect(
			new Set(
				filterOptions('projectSubtype', 'knitting', parents).map(
					(option) => option.id
				)
			)
		).toEqual(expected);
	});
	it('omits empty and unsupported parameters and preserves repeated filters', () => {
		expect(
			browseFilters(
				new URLSearchParams(
					'craft=sewing&status=&color=red&color=blue&color=red&projectType=&selfDrafted=false&private=ignored'
				)
			)
		).toEqual({ color: ['red', 'blue'] });
		expect(browseFilters(new URLSearchParams('selfDrafted=true'))).toEqual({
			selfDrafted: true
		});
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
		).toBe('/app/tags/knitting');
		for (const craft of [
			'knitting',
			'crochet',
			'sewing',
			'embroidery',
			'quilting'
		])
			expect(
				texture({ colour: 'cobalt', profileBackground: 'craft-' + craft })
			).toBe(`url('/profile_backgrounds/craft-${craft}.svg')`);
		expect(
			texture({ colour: 'cobalt', profileBackground: 'craft-../../private' })
		).toBeUndefined();
	});
});
