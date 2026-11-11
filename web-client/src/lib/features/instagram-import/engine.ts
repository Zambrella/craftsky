import {
	BlobReader,
	Writer,
	ZipReader,
	registerCodec,
	type FileEntry
} from '@zip.js/zip.js';
import { parseDetailedExport, parseLegacyExport } from './adapters';
import { mergeAdapterPosts } from './normalize';
import {
	classifyPostMetadataPath,
	isSafeArchivePath,
	resolveArchiveMediaPath
} from './paths';
import { inspectZipEnvelope } from './zipEnvelope';
import { inspectImageHeader } from './image-header';
import { Bzip2DecompressionStream } from '../settings/bzip-stream';
import { bzipLimits } from '../settings/bzip-limits';
registerCodec({
	compressionMethod: 12,
	format: 'bzip2',
	DecompressionStream: Bzip2DecompressionStream,
	versionNeeded: 46
});
import {
	IMPORT_SAFETY_POLICY as limits,
	withinCompressionRatio
} from './safety';
import { normalizeSourceTimestamp } from './timestamps';
import type {
	AdapterPost,
	InstagramArchive,
	InstagramPost,
	MediaSelection
} from './types';

/** Allocate only the declared bounded entry length; reject dishonest decompressed output. */
class BoundedWriter extends Writer<Uint8Array> {
	private bytes: Uint8Array<ArrayBuffer> = new Uint8Array();
	private offset = 0;
	constructor(private maximum: number) {
		super();
	}
	override async init(size = 0) {
		if (!Number.isSafeInteger(size) || size < 0 || size > this.maximum)
			throw new Error('entryLimit');
		this.bytes = new Uint8Array(size);
		await super.init?.(size);
	}
	override async writeUint8Array(chunk: Uint8Array) {
		if (
			this.offset + chunk.length > this.bytes.length ||
			this.offset + chunk.length > this.maximum
		)
			throw new Error('entryLimit');
		this.bytes.set(chunk, this.offset);
		this.offset += chunk.length;
	}
	override async getData() {
		if (this.offset !== this.bytes.length) throw new Error('invalidArchive');
		return this.bytes;
	}
}
function bounded(entry: FileEntry, maximum: number) {
	if (
		entry.compressionMethod === 12 &&
		(entry.uncompressedSize > bzipLimits.outputBytes ||
			entry.compressedSize > bzipLimits.compressedBytes)
	)
		throw new Error('entryLimit');
	if (
		!Number.isSafeInteger(entry.uncompressedSize) ||
		entry.uncompressedSize < 0 ||
		entry.uncompressedSize > maximum ||
		!withinCompressionRatio(
			entry.uncompressedSize,
			entry.compressedSize,
			limits.maxCompressionRatio
		)
	)
		throw new Error('entryLimit');
}
async function read(entry: FileEntry, maximum: number, signal?: AbortSignal) {
	signal?.throwIfAborted();
	bounded(entry, maximum);
	return entry.getData(new BoundedWriter(maximum), {
		signal,
		checkSignature: true,
		checkOverlappingEntry: true,
		preventClose: true,
		useWebWorkers: false
	});
}
function comparison(path: string) {
	return path.normalize('NFKC').toLocaleLowerCase('en-US');
}
function caption(input: string): string {
	if (
		!/(?:Ã.|Â.|â.|ð.|�)/.test(input) ||
		[...input].some((c) => c.charCodeAt(0) > 255)
	)
		return input;
	try {
		return new TextDecoder('utf-8', { fatal: true }).decode(
			Uint8Array.from([...input], (c) => c.charCodeAt(0))
		);
	} catch {
		return input;
	}
}
export async function sourceIdentity(
	createdAt: string,
	post: Pick<AdapterPost, 'media'>
): Promise<string> {
	const identity = [
		'instagram-import-post-v1',
		createdAt,
		...post.media.map((media) => `${media.kind}:${media.path.normalize('NFC')}`)
	].join('\0');
	const digest = await crypto.subtle.digest(
		'SHA-256',
		new TextEncoder().encode(identity)
	);
	return [...new Uint8Array(digest)]
		.map((byte) => byte.toString(16).padStart(2, '0'))
		.join('')
		.slice(0, 32);
}
function videoMime(bytes: Uint8Array): string | undefined {
	const tag = new TextDecoder('ascii').decode(bytes.subarray(4, 12));
	if (tag === 'ftypqt  ') return 'video/quicktime';
	if (['ftypisom', 'ftypmp41', 'ftypmp42', 'ftypavc1'].includes(tag))
		return 'video/mp4';
}

/** Runs in a dedicated worker in production. No archive or source payload is persisted or sent. */
export async function inspectInstagramArchive(
	file: Blob,
	signal?: AbortSignal,
	now = new Date()
): Promise<InstagramArchive> {
	const envelope = await inspectZipEnvelope(file);
	signal?.throwIfAborted();
	const reader = new ZipReader(new BlobReader(file), {
		useWebWorkers: false,
		strictness: 'strict',
		checkCrc32: true,
		checkOverlappingEntry: true
	});
	const entries = new Map<string, FileEntry>();
	const canonicalNames = new Map<string, string>();
	const ambiguous = new Set<string>();
	const candidates: {
		entry: FileEntry;
		wrapper: string;
		variant: 'legacy' | 'detailed';
	}[] = [];
	let count = 0;
	try {
		for await (const entry of reader.getEntriesGenerator()) {
			signal?.throwIfAborted();
			if (++count > envelope.entryCount || count > limits.maxEntries)
				throw new Error('entryCountLimit');
			if (
				entry.encrypted ||
				entry.symlink ||
				![0, 8, 9, 12].includes(entry.compressionMethod)
			)
				throw new Error('unsupportedArchive');
			if (entry.directory || !isSafeArchivePath(entry.filename)) continue;
			const key = comparison(entry.filename);
			const classified = classifyPostMetadataPath(entry.filename);
			if (canonicalNames.has(key)) {
				ambiguous.add(key);
				if (classified || classifyPostMetadataPath(canonicalNames.get(key)!))
					throw new Error('ambiguousArchive');
				continue;
			}
			canonicalNames.set(key, entry.filename);
			entries.set(entry.filename, entry);
			if (classified) candidates.push({ entry, ...classified });
		}
		if (
			count !== envelope.entryCount ||
			reader.directoryOffset !== envelope.centralDirectoryOffset ||
			reader.directoryLength !== envelope.centralDirectoryBytes
		)
			throw new Error('invalidArchive');
		if (!candidates.length) throw new Error('postsMissing');
		if (new Set(candidates.map((c) => c.wrapper)).size !== 1)
			throw new Error('ambiguousArchive');
		let total = 0;
		let skipped = 0;
		let recognized = 0;
		const parsed: AdapterPost[] = [];
		for (const candidate of candidates) {
			total += candidate.entry.uncompressedSize;
			if (total > limits.maxCombinedCandidateJsonBytes)
				throw new Error('metadataLimit');
			const data = JSON.parse(
				new TextDecoder('utf-8', { fatal: true }).decode(
					await read(candidate.entry, limits.maxCandidateJsonBytes, signal)
				)
			) as unknown;
			const result =
				candidate.variant === 'legacy'
					? parseLegacyExport(data)
					: parseDetailedExport(data);
			recognized += result.recognizedPostRecords;
			skipped += result.skipped.length;
			parsed.push(...result.posts);
			if (parsed.length + skipped > limits.maxPosts)
				throw new Error('postLimit');
		}
		if (!recognized) throw new Error('unsupportedPosts');
		const merged = mergeAdapterPosts(parsed);
		skipped += merged.skipped.length;
		const source = new Map<string, AdapterPost>();
		const posts: InstagramPost[] = [];
		for (const post of merged.posts) {
			signal?.throwIfAborted();
			const createdAt = normalizeSourceTimestamp(post.timestamp, now);
			if (!createdAt) {
				skipped++;
				continue;
			}
			const sourceKey = await sourceIdentity(createdAt, post);
			if (source.has(sourceKey)) throw new Error('ambiguousArchive');
			const imageCount = post.media.filter((m) => m.kind === 'image').length;
			const videoCount = post.media.filter((m) => m.kind === 'video').length;
			const text = caption(post.caption);
			const warnings: string[] = [];
			if (
				[...new Intl.Segmenter('en', { granularity: 'grapheme' }).segment(text)]
					.length > 2000 ||
				new TextEncoder().encode(text).length > 20000
			)
				warnings.push(
					'The caption exceeds the 2,000-character limit. Edit it before posting.'
				);
			if (imageCount > 8)
				warnings.push('Choose up to eight images for this post.');
			if (videoCount > 1) warnings.push('Choose one video for this post.');
			if (imageCount && videoCount)
				warnings.push(
					'Choose images or one video. Mixed carousels are not supported.'
				);
			if (post.media.some((m) => m.kind === 'unsupported'))
				warnings.push('Some media formats cannot be imported.');
			for (const media of post.media) {
				const path = resolveArchiveMediaPath(candidates[0].wrapper, media.path);
				const entry = entries.get(path);
				if (!entry || ambiguous.has(comparison(path))) {
					warnings.push('Some media files are missing or ambiguous.');
					break;
				}
				const max = media.kind === 'video' ? 300_000_000 : 40_000_000;
				try {
					bounded(entry, max);
				} catch {
					warnings.push(
						'Some media files exceed the supported size or archive safety limits.'
					);
					break;
				}
			}
			source.set(sourceKey, post);
			posts.push({
				sourceKey,
				createdAt,
				caption: text,
				imageCount,
				videoCount,
				warnings
			});
		}
		let closed = false;
		const media = async (
			post: InstagramPost,
			abort?: AbortSignal,
			selection?: MediaSelection
		) => {
			if (closed) throw new Error('archiveClosed');
			const original = source.get(post.sourceKey);
			if (!original) throw new Error('unknownPost');
			if (
				!selection &&
				((post.imageCount && post.videoCount) || post.videoCount > 1)
			)
				throw new Error('mediaChoiceRequired');
			const kind = selection?.kind ?? (post.videoCount ? 'video' : 'images');
			const candidatesMedia = original.media.filter(
				(m) => m.kind === (kind === 'video' ? 'video' : 'image')
			);
			const indices =
				selection?.indices ??
				candidatesMedia
					.slice(0, kind === 'video' ? 1 : 8)
					.map((_, index) => index);
			if (
				indices.length > (kind === 'video' ? 1 : 8) ||
				new Set(indices).size !== indices.length ||
				indices.some(
					(index) =>
						!Number.isInteger(index) ||
						index < 0 ||
						index >= candidatesMedia.length
				)
			)
				throw new Error('mediaChoiceRequired');
			const result: { images: File[]; videos: File[] } = {
				images: [],
				videos: []
			};
			for (const index of indices) {
				abort?.throwIfAborted();
				const chosen = candidatesMedia[index];
				const path = resolveArchiveMediaPath(
					candidates[0].wrapper,
					chosen.path
				);
				const entry = entries.get(path);
				if (!entry || ambiguous.has(comparison(path)))
					throw new Error('mediaMissing');
				const bytes = await read(
					entry,
					kind === 'video' ? 300_000_000 : 40_000_000,
					abort
				);
				const mime =
					kind === 'video' ? videoMime(bytes) : inspectImageHeader(bytes).mime;
				if (!mime) throw new Error('mediaFormat');
				// Use neutral file names; the archive path is not needed outside this worker.
				const extension =
					mime === 'image/jpeg'
						? 'jpg'
						: mime === 'image/png'
							? 'png'
							: mime === 'image/webp'
								? 'webp'
								: mime === 'video/quicktime'
									? 'mov'
									: 'mp4';
				const file = new File(
					[bytes as Uint8Array<ArrayBuffer>],
					`Instagram ${kind === 'video' ? 'video' : 'image'} ${index + 1}.${extension}`,
					{ type: mime }
				);
				(kind === 'video' ? result.videos : result.images).push(file);
			}
			return result;
		};
		return {
			posts,
			skipped,
			media,
			async preview(post, abort) {
				if (!post.imageCount) return;
				const selected = await media(post, abort, {
					kind: 'images',
					indices: [0]
				});
				const bitmap = await createImageBitmap(selected.images[0]);
				try {
					if (
						bitmap.width > limits.maxDimension ||
						bitmap.height > limits.maxDimension ||
						bitmap.width * bitmap.height > limits.maxDecodedPixels
					)
						return;
					const scale = Math.min(
						1,
						160 / Math.max(bitmap.width, bitmap.height)
					);
					const canvas = new OffscreenCanvas(
						Math.max(1, Math.round(bitmap.width * scale)),
						Math.max(1, Math.round(bitmap.height * scale))
					);
					canvas
						.getContext('2d')
						?.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
					return await canvas.convertToBlob({
						type: 'image/jpeg',
						quality: 0.75
					});
				} finally {
					bitmap.close();
				}
			},
			close() {
				closed = true;
				source.clear();
				entries.clear();
				void reader.close();
			}
		};
	} catch (error) {
		await reader.close().catch(() => {});
		throw error;
	}
}
