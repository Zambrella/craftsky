import { InputError } from '#lib/user-facing-error.ts';
import {
	BlobReader,
	ZipReader,
	Writer,
	registerCodec,
	type FileEntry
} from '@zip.js/zip.js';
import { parseFollowing, type ParsedFollowing } from './model.ts';
import { Bzip2DecompressionStream } from './bzip-stream.ts';
import { bzipLimits } from './bzip-limits.ts';

registerCodec({
	compressionMethod: 12,
	format: 'bzip2',
	DecompressionStream: Bzip2DecompressionStream,
	versionNeeded: 46
});

export const followingPath =
	'connections/followers_and_following/following.json';
const maximum = {
	fileBytes: 20 * 1024 * 1024,
	entries: 100000,
	directoryBytes: 64 * 1024 * 1024
};
type Limits = typeof maximum;
class ExportError extends InputError {}
const invalid = () =>
	new ExportError(
		'This Instagram archive is invalid or damaged. Extract following.json and try that file instead.'
	);
const tooLarge = () =>
	new ExportError(
		'The following list exceeds 20 MB. Request a smaller Instagram export.'
	);
const checkAbort = (signal?: AbortSignal) => {
	if (signal?.aborted)
		throw new DOMException('Import cancelled.', 'AbortError');
};
const integer = (value: number) => Number.isSafeInteger(value) && value >= 0;

async function range(
	file: Blob,
	offset: number,
	length: number,
	signal?: AbortSignal
): Promise<DataView> {
	checkAbort(signal);
	if (!integer(offset) || !integer(length) || offset + length > file.size)
		throw invalid();
	return new DataView(await file.slice(offset, offset + length).arrayBuffer());
}
function uint64(data: DataView, offset: number): number {
	const value = Number(data.getBigUint64(offset, true));
	if (!integer(value)) throw invalid();
	return value;
}

/** Preflight the small trailer before a library can allocate the central directory. */
async function metadata(file: Blob, limits: Limits, signal?: AbortSignal) {
	const tailOffset = Math.max(0, file.size - 65557),
		tail = await range(file, tailOffset, file.size - tailOffset, signal);
	let at = -1;
	for (let i = tail.byteLength - 22; i >= 0; i--) {
		if (
			tail.getUint32(i, true) === 0x06054b50 &&
			i + 22 + tail.getUint16(i + 20, true) === tail.byteLength
		) {
			at = i;
			break;
		}
	}
	if (at < 0) throw invalid();
	const eocd = tailOffset + at;
	let disk = tail.getUint16(at + 4, true),
		directoryDisk = tail.getUint16(at + 6, true),
		onDisk = tail.getUint16(at + 8, true),
		entries = tail.getUint16(at + 10, true),
		size = tail.getUint32(at + 12, true),
		offset = tail.getUint32(at + 16, true),
		metadataStart = eocd;
	if (
		entries === 0xffff ||
		onDisk === 0xffff ||
		size === 0xffffffff ||
		offset === 0xffffffff
	) {
		const locator = await range(file, eocd - 20, 20, signal);
		if (
			locator.getUint32(0, true) !== 0x07064b50 ||
			locator.getUint32(4, true) !== 0 ||
			locator.getUint32(16, true) !== 1
		)
			throw invalid();
		const start = uint64(locator, 8),
			zip64 = await range(file, start, 56, signal),
			recordSize = uint64(zip64, 4);
		if (
			zip64.getUint32(0, true) !== 0x06064b50 ||
			recordSize < 44 ||
			start + 12 + recordSize > eocd - 20
		)
			throw invalid();
		disk = zip64.getUint32(16, true);
		directoryDisk = zip64.getUint32(20, true);
		onDisk = uint64(zip64, 24);
		entries = uint64(zip64, 32);
		size = uint64(zip64, 40);
		offset = uint64(zip64, 48);
		metadataStart = start;
	}
	if (
		disk !== 0 ||
		directoryDisk !== 0 ||
		onDisk !== entries ||
		!integer(offset + size) ||
		offset + size > metadataStart
	)
		throw invalid();
	if (entries > limits.entries || size > limits.directoryBytes)
		throw new ExportError(
			'This archive has too many files or too much directory data. Extract following.json and import it directly.'
		);
	return { entries, offset, size };
}

class BoundedWriter extends Writer<Uint8Array<ArrayBuffer>> {
	private chunks: Uint8Array<ArrayBuffer>[] = [];
	private length = 0;
	constructor(private limit: number) {
		super();
	}
	override async writeUint8Array(value: Uint8Array) {
		if (this.length + value.byteLength > this.limit) throw tooLarge();
		this.length += value.byteLength;
		this.chunks.push(new Uint8Array(value));
	}
	override async getData() {
		const value = new Uint8Array(this.length);
		let offset = 0;
		for (const chunk of this.chunks) {
			value.set(chunk, offset);
			offset += chunk.byteLength;
		}
		this.chunks = [];
		return value;
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
	const signature = await range(file, 0, Math.min(4, file.size), signal);
	if (signature.byteLength < 4 || signature.getUint32(0, true) !== 0x04034b50) {
		if (file.size > limits.fileBytes) throw tooLarge();
		checkAbort(signal);
		return parseFollowing(await file.text(), true);
	}
	const info = await metadata(file, limits, signal);
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
			checkAbort(signal);
			if (++count > limits.entries) throw invalid();
			if (entry.symlink) throw invalid();
			if (entry.filename !== followingPath) continue;
			const raw = new TextDecoder('utf-8', { fatal: true }).decode(
				entry.rawFilename
			);
			if (target || entry.directory || raw !== followingPath || entry.encrypted)
				throw invalid();
			if (
				!integer(entry.uncompressedSize) ||
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
			count !== info.entries ||
			reader.directoryOffset !== info.offset ||
			reader.directoryLength !== info.size
		)
			throw invalid();
		if (!target)
			throw new ExportError(
				`The archive does not contain ${followingPath}. Choose an Instagram JSON export.`
			);
		const output = await target.getData(new BoundedWriter(limits.fileBytes), {
			signal
		});
		if (!output || output.byteLength !== target.uncompressedSize)
			throw invalid();
		return parseFollowing(
			new TextDecoder('utf-8', { fatal: true }).decode(output),
			true
		);
	} catch (error) {
		if (
			error instanceof ExportError ||
			(error instanceof DOMException && error.name === 'AbortError')
		)
			throw error;
		throw invalid();
	} finally {
		await reader.close();
	}
}
