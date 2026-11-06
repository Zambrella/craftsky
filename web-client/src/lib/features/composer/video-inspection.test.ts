import { afterEach, expect, it, vi } from 'vitest';
import { attachmentSelection, inspectVideo } from './media.ts';

afterEach(() => {
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});
function probe(duration: number, fails = false) {
	const element = {
		onloadedmetadata: null as null | (() => void),
		onerror: null as null | (() => void),
		duration,
		videoWidth: 1920,
		videoHeight: 1080,
		preload: '',
		set src(_value: string) {
			queueMicrotask(() =>
				fails ? this.onerror?.() : this.onloadedmetadata?.()
			);
		},
		removeAttribute: vi.fn(),
		load: vi.fn()
	};
	vi.stubGlobal('document', { createElement: () => element });
	vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:fixture');
	const revoke = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
	return { element, revoke };
}
function source(brand = 'qt  ') {
	return new File(
		[
			new Uint8Array([0, 0, 0, 20, ...new TextEncoder().encode('ftyp' + brand)])
		],
		'camera.MOV',
		{ type: '' }
	);
}
it('recognizes MOV without a supplied browser MIME and retains dimensions', async () => {
	const { element, revoke } = probe(600);
	expect(attachmentSelection([source()])).toBe('video');
	expect(await inspectVideo(source())).toMatchObject({
		mime: 'video/quicktime',
		width: 1920,
		height: 1080
	});
	expect(revoke).toHaveBeenCalledWith('blob:fixture');
	expect(element.load).toHaveBeenCalled();
});
it('allows a MOV that the browser cannot decode, with an explicit preview fallback', async () => {
	const { revoke } = probe(NaN, true);
	expect(await inspectVideo(source())).toMatchObject({
		previewUnavailable: true,
		mime: 'video/quicktime'
	});
	expect(revoke).toHaveBeenCalled();
});
it('rejects a known duration over ten minutes and releases the probe', async () => {
	const { revoke } = probe(600.01);
	await expect(inspectVideo(source())).rejects.toThrow('ten minutes');
	expect(revoke).toHaveBeenCalled();
});
it('keeps MP4 metadata validation and rejects forged MOV extensions', async () => {
	probe(NaN, true);
	await expect(inspectVideo(source('isom'))).rejects.toThrow(
		'cannot be previewed'
	);
	await expect(inspectVideo(new File(['invalid'], 'fake.mov'))).rejects.toThrow(
		'Choose an MP4 or MOV'
	);
});

it('rejects oversized videos before reading or probing them', async () => {
	const read = vi.fn();
	const file = { size: 300_000_001, slice: read } as unknown as File;
	await expect(inspectVideo(file)).rejects.toThrow(
		'This video is too large. Videos must be 300 MB or smaller.'
	);
	expect(read).not.toHaveBeenCalled();
});
