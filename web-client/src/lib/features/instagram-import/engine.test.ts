import { describe, expect, it } from 'vitest';
import {
	BlobWriter,
	TextReader,
	Uint8ArrayReader,
	ZipWriter
} from '@zip.js/zip.js';
import { inspectInstagramArchive, sourceIdentity } from './engine';
const timestamp = 1_700_000_000;
const date = '2023-11-14T22:13:20.000Z';
function png(width = 1, height = 1) {
	const bytes = new Uint8Array(45);
	bytes.set([137, 80, 78, 71, 13, 10, 26, 10]);
	const view = new DataView(bytes.buffer);
	view.setUint32(8, 13);
	bytes.set(new TextEncoder().encode('IHDR'), 12);
	view.setUint32(16, width);
	view.setUint32(20, height);
	bytes.set(new TextEncoder().encode('IDAT'), 37);
	return bytes;
}
const mp4 = new Uint8Array([
	0, 0, 0, 20, 102, 116, 121, 112, 105, 115, 111, 109, 0, 0, 0, 0, 105, 115,
	111, 109
]);
async function archive(
	posts: unknown,
	files: Record<string, Uint8Array> = {},
	name = 'your_instagram_activity/media/posts_1.json',
	zip64 = false,
	level = 0
) {
	const writer = new ZipWriter(new BlobWriter(), {
		useWebWorkers: false,
		zip64
	});
	await writer.add(name, new TextReader(JSON.stringify(posts)), { level });
	for (const [path, bytes] of Object.entries(files))
		await writer.add(path, new Uint8ArrayReader(bytes), { level });
	return writer.close();
}
const post = (paths = ['media/posts/one.png'], title = 'Caption') => ({
	title,
	creation_timestamp: timestamp,
	media: paths.map((uri) => ({ uri, creation_timestamp: timestamp }))
});
describe('local Instagram post archive', () => {
	it('preserves full captions and stable identity across edits, wrappers and ZIP64', async () => {
		const title = 'A'.repeat(2200);
		const first = await inspectInstagramArchive(
			await archive([post(undefined, title)])
		);
		const second = await inspectInstagramArchive(
			await archive(
				[post(undefined, 'edited')],
				{},
				'export/your_instagram_activity/media/posts_1.json',
				true
			)
		);
		try {
			expect(first.posts[0].caption).toBe(title);
			expect(first.posts[0].warnings).toContain(
				'The caption exceeds the 2,000-character limit. Edit it before posting.'
			);
			expect(first.posts[0].createdAt).toBe(date);
			expect(first.posts[0].sourceKey).toBe('3b8b90a45597db1f75313b11f40baddb');
			expect(first.posts[0].sourceKey).toBe(second.posts[0].sourceKey);
			expect(first.posts[0].sourceKey).toBe(
				await sourceIdentity(date, {
					media: [{ path: 'media/posts/one.png', kind: 'image' }]
				})
			);
		} finally {
			first.close();
			second.close();
		}
	});
	it('keeps video-only posts and extracts only after review', async () => {
		const missing = await inspectInstagramArchive(
			await archive([post(['media/posts/clip.mp4'])])
		);
		try {
			expect(missing.posts[0]).toMatchObject({ videoCount: 1, imageCount: 0 });
			await expect(missing.media(missing.posts[0])).rejects.toThrow(
				'mediaMissing'
			);
		} finally {
			missing.close();
		}
		const complete = await inspectInstagramArchive(
			await archive([post(['media/posts/clip.mp4'])], {
				'media/posts/clip.mp4': mp4
			})
		);
		try {
			const selected = await complete.media(complete.posts[0]);
			expect(selected.images).toHaveLength(0);
			expect(selected.videos[0].type).toBe('video/mp4');
			expect(new Uint8Array(await selected.videos[0].arrayBuffer())).toEqual(
				mp4
			);
		} finally {
			complete.close();
		}
	});
	it('reads reel JSON wrappers and keeps video paths', async () => {
		const result = await inspectInstagramArchive(
			await archive(
				{ ig_reels_media: [post(['media/reels/202311/clip.mp4'])] },
				{ 'media/reels/202311/clip.mp4': mp4 },
				'your_instagram_activity/media/reels.json'
			)
		);
		try {
			expect(result.posts[0].videoCount).toBe(1);
			expect((await result.media(result.posts[0])).videos).toHaveLength(1);
		} finally {
			result.close();
		}
	});
	it('requires choices for mixed/multiple videos and never extracts unselected missing media', async () => {
		const result = await inspectInstagramArchive(
			await archive(
				[
					post([
						'media/posts/one.png',
						'media/posts/missing.mp4',
						'media/posts/two.mp4'
					])
				],
				{ 'media/posts/one.png': png(), 'media/posts/two.mp4': mp4 }
			)
		);
		try {
			expect(result.posts[0]).toMatchObject({ imageCount: 1, videoCount: 2 });
			await expect(result.media(result.posts[0])).rejects.toThrow(
				'mediaChoiceRequired'
			);
			const images = await result.media(result.posts[0], undefined, {
				kind: 'images',
				indices: [0]
			});
			expect(images.images).toHaveLength(1);
			expect(images.videos).toHaveLength(0);
			expect(
				(
					await result.media(result.posts[0], undefined, {
						kind: 'video',
						indices: [1]
					})
				).videos
			).toHaveLength(1);
			await expect(
				result.media(result.posts[0], undefined, {
					kind: 'video',
					indices: [0, 1]
				})
			).rejects.toThrow('mediaChoiceRequired');
		} finally {
			result.close();
		}
	});
	it('lets users choose any eight photos and preserves the original count', async () => {
		const paths = Array.from({ length: 10 }, (_, i) => `media/posts/${i}.png`);
		const result = await inspectInstagramArchive(
			await archive([post(paths)], { 'media/posts/9.png': png() })
		);
		try {
			expect(result.posts[0].imageCount).toBe(10);
			expect(result.posts[0].warnings).toContain(
				'Choose up to eight images for this post.'
			);
			expect(
				(
					await result.media(result.posts[0], undefined, {
						kind: 'images',
						indices: [9]
					})
				).images
			).toHaveLength(1);
			await expect(
				result.media(result.posts[0], undefined, {
					kind: 'images',
					indices: [0, 1, 2, 3, 4, 5, 6, 7, 8]
				})
			).rejects.toThrow('mediaChoiceRequired');
		} finally {
			result.close();
		}
	});
	it('rejects unsupported image contents without exposing paths or content', async () => {
		const result = await inspectInstagramArchive(
			await archive([post()], {
				'media/posts/one.png': new TextEncoder().encode(
					'PRIVATE-PAYLOAD-CANARY'
				)
			})
		);
		try {
			await expect(result.media(result.posts[0])).rejects.toThrow(
				'unsupportedImageFormat'
			);
		} finally {
			result.close();
		}
	});
	it('rejects oversized image headers before decoding', async () => {
		const result = await inspectInstagramArchive(
			await archive([post()], { 'media/posts/one.png': png(12000, 12000) })
		);
		try {
			await expect(result.media(result.posts[0])).rejects.toThrow(
				'imagePixelLimit'
			);
		} finally {
			result.close();
		}
	});
	it('ignores traversal references and rejects ambiguous archive paths', async () => {
		const result = await inspectInstagramArchive(
			await archive([post(['media/posts/../private.png'])])
		);
		try {
			expect(result.posts).toHaveLength(0);
			expect(result.skipped).toBe(1);
		} finally {
			result.close();
		}
		await expect(
			inspectInstagramArchive(
				await archive([post()], {
					'your_instagram_activity/media/POSTS_1.json':
						new TextEncoder().encode('[]')
				})
			)
		).rejects.toThrow('ambiguousArchive');
	});
	it('merges equivalent records and skips invalid dates', async () => {
		const result = await inspectInstagramArchive(
			await archive([
				post(),
				post(),
				{
					...post(['media/posts/old.png']),
					creation_timestamp: 0,
					media: [{ uri: 'media/posts/old.png', creation_timestamp: 0 }]
				}
			])
		);
		try {
			expect(result.posts).toHaveLength(1);
			expect(result.skipped).toBe(1);
		} finally {
			result.close();
		}
	});
	it('rejects corrupt media metadata by CRC', async () => {
		const original = new Uint8Array(
			await (await archive([post()])).arrayBuffer()
		);
		const view = new DataView(original.buffer);
		const offset = 30 + view.getUint16(26, true) + view.getUint16(28, true);
		original[offset + 3] ^= 1;
		await expect(
			inspectInstagramArchive(new Blob([original]))
		).rejects.toThrow();
	});
	it('supports compressed exports and refuses use after closing or abort', async () => {
		const result = await inspectInstagramArchive(
			await archive([post()], {}, undefined, false, 8)
		);
		result.close();
		await expect(result.media(result.posts[0])).rejects.toThrow(
			'archiveClosed'
		);
		const controller = new AbortController();
		controller.abort();
		await expect(
			inspectInstagramArchive(await archive([post()]), controller.signal)
		).rejects.toThrow();
	});
});
