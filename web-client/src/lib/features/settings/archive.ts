import { InputError } from '#lib/user-facing-error.ts';
import {
	BlobReader,
	ZipReader,
	registerCodec,
	type FileEntry
} from '@zip.js/zip.js';
import { parseFollowing, type ParsedFollowing } from './model.ts';
import { Bzip2DecompressionStream } from './bzip-stream.ts';
import { bzipLimits } from './bzip-limits.ts';
import { IMPORT_SAFETY_POLICY } from '../instagram-import/safety';
import {
	inspectZipEnvelope,
	type ZipEnvelope
} from '../instagram-import/zipEnvelope';
import { ExactZipEntryWriter } from '../instagram-import/zip-entry-writer';

registerCodec({
	compressionMethod: 12,
	format: 'bzip2',
	DecompressionStream: Bzip2DecompressionStream,
	versionNeeded: 46
});

export const followingPath =
	'connections/followers_and_following/following.json';
type Limits = { fileBytes: number; entries: number; directoryBytes: number };
const maximum: Limits = {
	fileBytes: 20 * 1024 * 1024,
	entries: IMPORT_SAFETY_POLICY.maxEntries,
	directoryBytes: IMPORT_SAFETY_POLICY.maxCentralDirectoryBytes
};
class ExportError extends InputError {}
const invalid = () =>
	new ExportError(
		'This Instagram archive is invalid or damaged. Extract following.json and try that file instead.'
	);
const tooLarge = () =>
	new ExportError(
		'The following list exceeds 20 MB. Request a smaller Instagram export.'
	);
async function inspectFollowingZip(
	file: Blob,
	limits: Limits
): Promise<ZipEnvelope> {
	try {
		return await inspectZipEnvelope(file, {
			maxEntries: limits.entries,
			maxCentralDirectoryBytes: limits.directoryBytes
		});
	} catch (error) {
		if (
			error instanceof Error &&
			['archiveEntryLimit', 'centralDirectoryLimit'].includes(error.message)
		)
			throw new ExportError(
				'This archive has too many files or too much directory data. Extract following.json and import it directly.'
			);
		throw invalid();
	}
}

/** Reads only the exact following.json entry. Nothing is extracted to disk or sent to a server. */
export async function parseInstagramExport(
	file: Blob,
	signal?: AbortSignal,
	testLimits: Partial<Limits> = {}
): Promise<ParsedFollowing> {
	const limits = Object.fromEntries(
		Object.entries(maximum).map(([key, value]) => [
			key,
			Math.max(1, Math.min(value, testLimits[key as keyof Limits] ?? value))
		])
	) as Limits;
	signal?.throwIfAborted();
	const signature = new DataView(await file.slice(0, 4).arrayBuffer());
	signal?.throwIfAborted();
	if (signature.byteLength < 4 || signature.getUint32(0, true) !== 0x04034b50) {
		if (file.size > limits.fileBytes) throw tooLarge();
		const data = await file.arrayBuffer();
		signal?.throwIfAborted();
		let source: string;
		try {
			source = new TextDecoder('utf-8', { fatal: true }).decode(data);
		} catch {
			throw new InputError(
				'This file is not valid UTF-8. Choose Instagram’s following.json file.'
			);
		}
		return parseFollowing(source);
	}
	const info = await inspectFollowingZip(file, limits);
	signal?.throwIfAborted();
	const reader = new ZipReader(new BlobReader(file), {
		strictness: 'strict',
		checkCrc32: true,
		checkOverlappingEntry: true,
		useWebWorkers: false,
		useCompressionStream: true,
		signal
	});
	try {
		let count = 0,
			target: FileEntry | undefined;
		for await (const entry of reader.getEntriesGenerator()) {
			signal?.throwIfAborted();
			if (++count > info.entryCount) throw invalid();
			if (entry.symlink) throw invalid();
			if (entry.filename !== followingPath) continue;
			const raw = new TextDecoder('utf-8', { fatal: true }).decode(
				entry.rawFilename
			);
			if (target || entry.directory || raw !== followingPath || entry.encrypted)
				throw invalid();
			if (
				!Number.isSafeInteger(entry.uncompressedSize) ||
				entry.uncompressedSize < 0 ||
				entry.uncompressedSize > limits.fileBytes
			)
				throw tooLarge();
			if (
				entry.compressionMethod === 12 &&
				entry.compressedSize > bzipLimits.compressedBytes
			)
				throw invalid();
			if (![0, 8, 9, 12].includes(entry.compressionMethod))
				throw new ExportError(
					'This archive uses unsupported compression. Extract following.json and import that file instead.'
				);
			target = entry;
		}
		if (
			count !== info.entryCount ||
			reader.directoryOffset !== info.centralDirectoryOffset ||
			reader.directoryLength !== info.centralDirectoryBytes
		)
			throw invalid();
		if (!target)
			throw new ExportError(
				`The archive does not contain ${followingPath}. Choose an Instagram JSON export.`
			);
		const output = await target.getData(
			new ExactZipEntryWriter(limits.fileBytes),
			{
				signal
			}
		);
		signal?.throwIfAborted();
		return parseFollowing(
			new TextDecoder('utf-8', { fatal: true }).decode(output)
		);
	} catch (error) {
		if (error instanceof InputError || signal?.aborted) throw error;
		throw invalid();
	} finally {
		await reader.close();
	}
}
