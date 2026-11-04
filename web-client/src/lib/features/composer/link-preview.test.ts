import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	ComposerLinkPreviews,
	previewCandidates,
	previewNavigation,
	type PreviewState,
	type ComposerPreview,
	type PreviewCandidate
} from './link-preview';
const preview = (url: string): ComposerPreview => ({
	external: { uri: url, title: new URL(url).hostname, description: 'Details' }
});
const empty = (): PreviewState => ({
	index: 0,
	total: 0,
	loading: false,
	dismissed: false,
	suppressed: false
});
afterEach(() => vi.useRealTimers());
describe('native automatic link preview candidates', () => {
	it('requires a completion boundary and ignores mentions, email and credential URLs', () => {
		expect(
			previewCandidates(
				'First https://one.example/path then https://unfinished.example'
			)
		).toEqual([{ identity: 'https://one.example/path', fragment: '' }]);
		expect(
			previewCandidates(
				'@maker.example maker@example.com https://user:secret@example.com/path '
			)
		).toEqual([]);
		expect(
			previewCandidates(
				'https://one.example/path\nhttps://two.example/path, next'
			).map((value) => value.identity)
		).toEqual(['https://one.example/path', 'https://two.example/path']);
	});
	it('deduplicates canonical identities, strips fragments from transport, and caps fetch candidates at four', () => {
		const candidates = previewCandidates(
			'example.com/pattern#first https://EXAMPLE.COM:443/pattern#second other.example/path three.example/path four.example/path five.example/path '
		);
		expect(candidates).toHaveLength(4);
		expect(candidates[0]).toEqual({
			identity: 'https://example.com/pattern',
			fragment: '#first'
		});
		expect(
			previewNavigation(candidates[0], 'https://final.example/pattern')
		).toBe('https://final.example/pattern#first');
		expect(
			previewNavigation(
				candidates[0],
				'https://final.example/pattern#canonical'
			)
		).toBe('https://final.example/pattern#canonical');
		expect(() =>
			previewNavigation(candidates[0], 'javascript:alert(1)')
		).toThrow();
	});
	it('retains a seeded saved candidate without requiring new trailing whitespace', () => {
		expect(
			previewCandidates('https://one.example/path', 'https://one.example/path')
		).toHaveLength(1);
	});
});
describe('activation-owned preview requests', () => {
	it('debounces typed changes and automatically chooses the earliest successful candidate', async () => {
		vi.useFakeTimers();
		let state = empty();
		const fetcher = vi.fn(async (candidate: PreviewCandidate) =>
			preview(candidate.identity)
		);
		const controller = new ComposerLinkPreviews(
			fetcher,
			(value) => (state = value)
		);
		controller.update('one.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(150);
		controller.update('two.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(299);
		expect(fetcher).not.toHaveBeenCalled();
		await vi.advanceTimersByTimeAsync(1);
		expect(fetcher).toHaveBeenCalledOnce();
		expect(state.selected?.external.uri).toBe('https://two.example/path');
	});
	it('fetches sequentially, skips a failed URL, and moves only through current successful previews', async () => {
		vi.useFakeTimers();
		let state = empty();
		const fetcher = vi.fn(async (candidate: PreviewCandidate) => {
			if (candidate.identity.includes('bad')) throw new Error('Unavailable');
			return preview(candidate.identity);
		});
		const controller = new ComposerLinkPreviews(
			fetcher,
			(value) => (state = value)
		);
		controller.update(
			'bad.example/path good.example/path last.example/path ',
			false,
			false
		);
		await vi.advanceTimersByTimeAsync(900);
		expect(fetcher).toHaveBeenCalledTimes(3);
		expect(state.total).toBe(2);
		expect(state.index).toBe(1);
		expect(state.selected?.external.uri).toBe('https://good.example/path');
		controller.move(1);
		expect(state.index).toBe(2);
		expect(state.selected?.external.uri).toBe('https://last.example/path');
		controller.move(1);
		expect(state.index).toBe(1);
	});
	it('suppresses media previews, preserves explicit dismissal across text changes, and restores cached choices on Undo', async () => {
		vi.useFakeTimers();
		let state = empty();
		const fetcher = vi.fn(async (candidate: PreviewCandidate) =>
			preview(candidate.identity)
		);
		const controller = new ComposerLinkPreviews(
			fetcher,
			(value) => (state = value)
		);
		controller.update('one.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(300);
		expect(state.selected).toBeTruthy();
		controller.update('one.example/path ', true, false);
		expect(state.selected).toBeUndefined();
		controller.update('one.example/path ', false, false);
		expect(state.selected).toBeTruthy();
		expect(fetcher).toHaveBeenCalledOnce();
		controller.update('one.example/path ', false, true);
		controller.update('one.example/path two.example/path ', false, true);
		await vi.advanceTimersByTimeAsync(1000);
		expect(fetcher).toHaveBeenCalledOnce();
		expect(state.selected).toBeUndefined();
		controller.update('one.example/path two.example/path ', false, false);
		expect(state.selected?.external.uri).toBe('https://one.example/path');
	});
	it('aborts and ignores a late result from a removed URL and from a disposed account', async () => {
		vi.useFakeTimers();
		let state = empty();
		let finish!: (value: ComposerPreview) => void;
		let signal!: AbortSignal;
		const fetcher = vi.fn(
			(_candidate: PreviewCandidate, ownedSignal: AbortSignal) => {
				signal = ownedSignal;
				return new Promise<ComposerPreview>((resolve) => (finish = resolve));
			}
		);
		const publish = vi.fn((value: PreviewState) => (state = value)),
			controller = new ComposerLinkPreviews(fetcher, publish);
		controller.update('old.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(300);
		controller.update('No URL', false, false);
		expect(signal.aborted).toBe(true);
		finish(preview('https://old.example/path'));
		await vi.runAllTimersAsync();
		expect(state.selected).toBeUndefined();
		controller.update('new.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(300);
		controller.dispose();
		publish.mockClear();
		finish(preview('https://new.example/path'));
		await vi.runAllTimersAsync();
		expect(publish).not.toHaveBeenCalled();
	});
	it('freezes pending fetches at submit and does not attach a late preview to the frozen publication', async () => {
		vi.useFakeTimers();
		let state = empty();
		let finish!: (value: ComposerPreview) => void;
		const fetcher = vi.fn(
			() => new Promise<ComposerPreview>((resolve) => (finish = resolve))
		);
		const controller = new ComposerLinkPreviews(
			fetcher,
			(value) => (state = value)
		);
		controller.update('one.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(300);
		controller.pause();
		finish(preview('https://one.example/path'));
		await vi.runAllTimersAsync();
		expect(state.selected).toBeUndefined();
		controller.update('one.example/path ', false, false);
		await vi.advanceTimersByTimeAsync(1000);
		expect(fetcher).toHaveBeenCalledOnce();
	});
});
