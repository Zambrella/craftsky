import { describe, it, expect } from 'vitest';
import { BlobWriter, TextReader, ZipWriter } from '@zip.js/zip.js';
import { parseInstagramExport, followingPath } from './archive.ts';

const json = JSON.stringify({
	relationships_following: [{ string_list_data: [{ value: 'maker.one' }] }]
});
async function archive(
	files: { name: string; value: string }[] = [
		{ name: followingPath, value: json }
	],
	zip64 = false,
	level = 0
) {
	const writer = new ZipWriter(new BlobWriter(), {
		useWebWorkers: false,
		useCompressionStream: true,
		zip64,
		dataDescriptor: false
	});
	for (const file of files)
		await writer.add(file.name, new TextReader(file.value), { level });
	return writer.close();
}
function central(bytes: Uint8Array) {
	const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
	for (let i = 0; i < bytes.length - 4; i++)
		if (view.getUint32(i, true) === 0x02014b50) return i;
	throw new Error('Fixture missing directory');
}
describe('Instagram archive import', () => {
	it('reads plain JSON and extracts only the exact following list from an archive', async () => {
		expect((await parseInstagramExport(new Blob([json]))).entries).toEqual([
			{ username: 'maker.one' }
		]);
		const zip = await archive([
			{ name: 'messages/private.txt', value: 'Do not read this' },
			{ name: followingPath, value: json }
		]);
		expect((await parseInstagramExport(zip)).entries).toEqual([
			{ username: 'maker.one' }
		]);
	});
	it('supports stored, compressed and ZIP64 Instagram exports', async () => {
		for (const [zip64, level] of [
			[false, 8],
			[true, 0]
		] as const)
			expect(
				(await parseInstagramExport(await archive(undefined, zip64, level)))
					.entries
			).toHaveLength(1);
	});
	it('enforces central-directory and entry-count limits before decompression', async () => {
		const zip = await archive([
			{ name: 'extra', value: 'x' },
			{ name: followingPath, value: json }
		]);
		await expect(
			parseInstagramExport(zip, undefined, { entries: 1 })
		).rejects.toThrow('too many files');
		await expect(
			parseInstagramExport(zip, undefined, { directoryBytes: 1 })
		).rejects.toThrow('directory data');
		await expect(
			parseInstagramExport(zip, undefined, { fileBytes: 10 })
		).rejects.toThrow('20 MB');
	});
	it('rejects corrupt content using CRC32', async () => {
		const zip = new Uint8Array(await (await archive()).arrayBuffer()),
			view = new DataView(zip.buffer);
		const data = 30 + view.getUint16(26, true) + view.getUint16(28, true);
		zip[data + 10] ^= 1;
		await expect(parseInstagramExport(new Blob([zip]))).rejects.toThrow(
			'invalid or damaged'
		);
	});
	it('rejects a different local filename and symlinks', async () => {
		const original = new Uint8Array(await (await archive()).arrayBuffer());
		const mismatched = original.slice();
		mismatched[30] = 'X'.charCodeAt(0);
		await expect(parseInstagramExport(new Blob([mismatched]))).rejects.toThrow(
			'invalid or damaged'
		);
		const symlink = original.slice(),
			view = new DataView(symlink.buffer),
			at = central(symlink);
		view.setUint16(at + 4, 0x0314, true);
		view.setUint32(at + 38, (0xa1ff << 16) >>> 0, true);
		await expect(parseInstagramExport(new Blob([symlink]))).rejects.toThrow(
			'invalid or damaged'
		);
	});
	it('bounds actual decompressed output even when both header sizes lie', async () => {
		const zip = new Uint8Array(await (await archive()).arrayBuffer()),
			view = new DataView(zip.buffer);
		view.setUint32(22, 1, true);
		view.setUint32(central(zip) + 24, 1, true);
		await expect(
			parseInstagramExport(new Blob([zip]), undefined, { fileBytes: 10 })
		).rejects.toThrow();
	});
	it('rejects missing expected paths and an aborted read', async () => {
		await expect(
			parseInstagramExport(
				await archive([{ name: 'following.json', value: json }])
			)
		).rejects.toThrow('does not contain');
		const controller = new AbortController();
		controller.abort();
		await expect(
			parseInstagramExport(new Blob([json]), controller.signal)
		).rejects.toMatchObject({ name: 'AbortError' });
	});
});
