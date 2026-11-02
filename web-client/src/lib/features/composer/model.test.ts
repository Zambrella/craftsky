import { describe, expect, it } from 'vitest';
import {
	buildFacets,
	editChanges,
	freezeWrite,
	options,
	postPayload,
	validateDraft,
	textLength
} from './model.ts';
import { newDraft } from '../drafts/store.ts';
import type { JsonObject, PostEditSeed } from '#lib/api/client.ts';

describe('composer data contracts', () => {
	it('builds UTF-8 facet offsets after emoji and resolves exact mention handles', async () => {
		const text = '🧶 Hello @maker.example #knitting https://example.com';
		const handles: string[] = [];
		const facets = await buildFacets(text, async (handle) => {
			handles.push(handle);
			return 'did:plc:maker';
		});
		expect(handles).toEqual(['maker.example']);
		const encoded = new TextEncoder().encode(text);
		const decoder = new TextDecoder();
		expect(
			facets.map((facet) =>
				decoder.decode(
					encoded.slice(facet.index.byteStart, facet.index.byteEnd)
				)
			)
		).toEqual(['@maker.example', '#knitting', 'https://example.com']);
		expect(facets[0].features[0]).toEqual({
			$type: 'app.bsky.richtext.facet#mention',
			did: 'did:plc:maker'
		});
	});
	it('leaves unresolved mentions as readable text while keeping independent hashtags', async () => {
		const facets = await buildFacets('@missing.example #sewing', async () => {
			throw new Error('Not found');
		});
		expect(facets).toHaveLength(1);
		expect(facets[0].features[0].tag).toBe('sewing');
	});
	it('uses the canonical Flutter taxonomy and craft-specific subtype inventory', () => {
		expect(options('craft').map((item) => item.id)).toContain(
			'social.craftsky.feed.defs#knitting'
		);
		const sewing = options(
			'projectSubtype',
			'social.craftsky.feed.defs#sewing'
		);
		expect(sewing.length).toBeGreaterThan(20);
		expect(
			sewing.every(
				(item) =>
					item.craft === null ||
					item.craft === 'social.craftsky.feed.defs#sewing'
			)
		).toBe(true);
	});
	it('validates grapheme/UTF-8 limits and disallowed scheduled media shapes', () => {
		const draft = newDraft('did:plc:maker');
		draft.text = '🧶'.repeat(2000);
		expect(textLength(draft.text)).toBe(2000);
		expect(validateDraft(draft)).toEqual([]);
		draft.text += '!';
		expect(validateDraft(draft)).toContain(
			'Posts can contain up to 2,000 characters.'
		);
		draft.text = 'Hello';
		draft.video = {
			id: 'video',
			name: 'video.mp4',
			mime: 'video/mp4',
			alt: ''
		};
		expect(validateDraft(draft, true)).toContain(
			'Only new posts and projects with images or links can be scheduled.'
		);
	});
	it('freezes the exact keyed payload independently of subsequent editor changes', () => {
		const draft = newDraft('did:plc:maker');
		draft.text = 'Before';
		const body: JsonObject = {
			text: 'Before',
			langs: ['en'],
			sponsored: false
		};
		const frozen = freezeWrite(body, draft);
		body.text = 'After';
		(body.langs as string[]).push('fr');
		expect(frozen.body).toEqual({
			text: 'Before',
			langs: ['en'],
			sponsored: false
		});
		expect(frozen.key).toMatch(/^[0-9a-f-]{36}$/);
		expect(frozen.kind).toBe('create');
	});
	it('keeps unmodified media and project fields out of an edit and preserves retained image indices when reordered', () => {
		const draft = newDraft('did:plc:maker');
		draft.text = 'Before';
		draft.edit = {
			editable: {
				text: 'Before',
				langs: ['en'],
				sponsored: false,
				images: [
					{ existingIndex: 0, alt: 'First' },
					{ existingIndex: 1, alt: 'Second' }
				]
			},
			post: { author: { did: draft.owner }, rkey: 'post' },
			baseCid: 'cid'
		} as unknown as PostEditSeed;
		draft.images = [
			{
				id: '1',
				name: 'First',
				mime: 'image/jpeg',
				existingIndex: 0,
				alt: 'First'
			},
			{
				id: '2',
				name: 'Second',
				mime: 'image/jpeg',
				existingIndex: 1,
				alt: 'Second'
			}
		];
		expect(editChanges(draft, [])).toEqual({});
		draft.images.reverse();
		draft.images[0].alt = 'Updated';
		expect(editChanges(draft, [])).toEqual({
			images: [
				{ existingIndex: 1, alt: 'Updated' },
				{ existingIndex: 0, alt: 'First' }
			]
		});
		draft.text = 'Changed';
		expect(editChanges(draft, [])).toMatchObject({
			text: 'Changed',
			facets: null
		});
	});
	it('requires completed uploads and never puts local blob bytes in post JSON', () => {
		const draft = newDraft('did:plc:maker');
		draft.text = 'A quilt';
		draft.images = [
			{
				id: '1',
				name: 'quilt.jpg',
				mime: 'image/jpeg',
				blob: new Blob(['private']),
				alt: 'A blue quilt'
			}
		];
		expect(() => postPayload(draft, [])).toThrow('not finished');
		draft.images[0].uploaded = {
			$type: 'blob',
			ref: { $link: 'bafy-image' },
			mimeType: 'image/jpeg',
			size: 7
		};
		const payload = postPayload(draft, []);
		expect(payload.images?.[0]).toEqual({
			image: draft.images[0].uploaded,
			alt: 'A blue quilt'
		});
		expect(JSON.stringify(payload)).not.toContain('private');
	});
	it('updates external metadata and retained video alt text using partial edit objects', () => {
		const draft = newDraft('did:plc:maker');
		draft.text = 'Link';
		draft.edit = {
			editable: {
				text: 'Link',
				langs: ['en'],
				sponsored: false,
				embed: {
					$type: 'app.bsky.embed.external',
					external: {
						uri: 'https://example.com',
						title: 'Before',
						description: 'Details',
						thumb: {
							$type: 'blob',
							ref: { $link: 'old-thumb' },
							mimeType: 'image/jpeg',
							size: 10
						}
					}
				}
			}
		} as unknown as PostEditSeed;
		draft.external = {
			uri: 'https://example.com',
			title: 'After',
			description: 'Details',
			thumb: {
				$type: 'blob',
				ref: { $link: 'old-thumb' },
				mimeType: 'image/jpeg',
				size: 10
			}
		};
		expect(editChanges(draft, [])).toEqual({
			embed: { external: { title: 'After' } }
		});
		draft.edit.editable.embed = {
			$type: 'app.bsky.embed.video',
			alt: 'Original description'
		};
		draft.external = undefined;
		draft.embedChanges = { alt: 'Updated description' };
		expect(editChanges(draft, [])).toEqual({
			embed: { alt: 'Updated description' }
		});
	});
});
