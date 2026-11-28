import {
	BlobReader,
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
import { ExactZipEntryWriter } from './zip-entry-writer';
import { postLimits, videoSourceMime } from '../composer/limits';
import { graphemeBoundary } from '#lib/text.ts';
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
	InstagramArchiveSource,
	InstagramPost,
	MediaSelection
} from './types';

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
	return entry.getData(new ExactZipEntryWriter(maximum), {
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
	if (!/(?:Ã.|Â.|â.|ð.)/.test(input) || /[\u0100-\uffff]/.test(input))
		return input;
	const bytes = new Uint8Array(input.length);
	for (let i = 0; i < input.length; i++) bytes[i] = input.charCodeAt(i);
	try {
		return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
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
/** Runs in a dedicated worker in production. No archive or source payload is persisted or sent. */
export async function inspectInstagramArchive(
	file: Blob,
	signal?: AbortSignal,
	now = new Date()
): Promise<InstagramArchiveSource> {
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
		const mediaEntry = (path: string) => {
			const resolved = resolveArchiveMediaPath(candidates[0].wrapper, path);
			if (!ambiguous.has(comparison(resolved))) return entries.get(resolved);
		};
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
			if (parsed.length + result.posts.length + skipped > limits.maxPosts)
				throw new Error('postLimit');
			for (const post of result.posts) parsed.push(post);
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
				text.length > postLimits.text.bytes ||
				graphemeBoundary(text, postLimits.text.graphemes) < text.length ||
				new TextEncoder().encode(text).length > postLimits.text.bytes
			)
				warnings.push(
					'The caption exceeds the 2,000-character limit. Edit it before posting.'
				);
			if (imageCount > postLimits.imageCount)
				warnings.push('Choose up to eight images for this post.');
			if (videoCount > 1) warnings.push('Choose one video for this post.');
			if (imageCount && videoCount)
				warnings.push(
					'Choose images or one video. Mixed carousels are not supported.'
				);
			if (post.media.some((m) => m.kind === 'unsupported'))
				warnings.push('Some media formats cannot be imported.');
			for (const media of post.media) {
				const entry = mediaEntry(media.path);
				if (!entry) {
					warnings.push('Some media files are missing or ambiguous.');
					break;
				}
				const max =
					media.kind === 'video'
						? postLimits.videoSourceBytes
						: postLimits.imageSourceBytes;
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
		const readMedia = async (
			entry: FileEntry,
			kind: 'image' | 'video',
			index: number,
			abort?: AbortSignal
		) => {
			const bytes = await read(
				entry,
				kind === 'video'
					? postLimits.videoSourceBytes
					: postLimits.imageSourceBytes,
				abort
			);
			abort?.throwIfAborted();
			const mime =
				kind === 'video'
					? videoSourceMime(bytes)
					: inspectImageHeader(bytes).mime;
			if (!mime) throw new Error('mediaFormat');
			// Use neutral file names; the archive path is not needed outside this worker.
			const extension =
				mime === 'image/jpeg'
					? 'jpg'
					: mime === 'video/quicktime'
						? 'mov'
						: mime.split('/')[1];
			return new File([bytes], `Instagram ${kind} ${index + 1}.${extension}`, {
				type: mime
			});
		};
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
					.slice(0, kind === 'video' ? 1 : postLimits.imageCount)
					.map((_, index) => index);
			if (
				indices.length > (kind === 'video' ? 1 : postLimits.imageCount) ||
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
				const entry = mediaEntry(candidatesMedia[index].path);
				if (!entry) throw new Error('mediaMissing');
				const file = await readMedia(
					entry,
					kind === 'video' ? 'video' : 'image',
					index,
					abort
				);
				(kind === 'video' ? result.videos : result.images).push(file);
			}
			return result;
		};
		return {
			posts,
			skipped,
			media,
			async previewSource(post, abort) {
				if (closed) throw new Error('archiveClosed');
				const original = source.get(post.sourceKey);
				if (!original) throw new Error('unknownPost');
				for (const kind of ['image', 'video'] as const) {
					const maxAttempts = kind === 'image' ? postLimits.imageCount : 1;
					let attempts = 0;
					for (const [index, attachment] of original.media.entries()) {
						abort?.throwIfAborted();
						if (attachment.kind !== kind) continue;
						const entry = mediaEntry(attachment.path);
						if (!entry) continue;
						if (attempts === maxAttempts) break;
						attempts++;
						try {
							return await readMedia(entry, kind, index, abort);
						} catch {
							abort?.throwIfAborted();
						}
					}
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
