import { afterEach, expect, it, vi } from 'vitest';
import type { Video } from '#lib/api/client.ts';
import { bunnyEmbedURL, connectBunnyPlayer } from './bunny-player.ts';
const embed =
	'https://player.mediadelivery.net/embed/123/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa';
const video = (embedUrl = embed) =>
	({ embedUrl, playlist: 'https://media.test/video.m3u8' }) as Video;
afterEach(() => {
	vi.unstubAllGlobals();
	vi.useRealTimers();
});
it('selects only explicit trusted Bunny embeds and retains PDS caption playback', () => {
	expect(bunnyEmbedURL(video())).toBe(embed);
	expect(
		bunnyEmbedURL({
			...video(),
			captions: [{ cid: 'caption', lang: 'en', uri: '/caption' }]
		})
	).toBeUndefined();
	for (const value of [
		'https://foreign.test/embed/123/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa',
		embed + '?token=secret',
		embed + '#fragment',
		embed.replace('https:', 'http:'),
		embed.replace('https://', 'https://user:pass@'),
		embed.replace('/123/', '/not-a-library/'),
		'invalid'
	])
		expect(bunnyEmbedURL(video(value))).toBeUndefined();
	expect(bunnyEmbedURL({ playlist: embed } as Video)).toBeUndefined();
});
it('scopes the protocol to the exact iframe and disposes feed lifecycle listeners', () => {
	vi.useFakeTimers();
	let observe!: (entries: { isIntersecting: boolean }[]) => void;
	const disconnect = vi.fn();
	vi.stubGlobal(
		'IntersectionObserver',
		class {
			constructor(callback: typeof observe) {
				observe = callback;
			}
			observe = vi.fn();
			disconnect = disconnect;
		}
	);
	const scope = { addEventListener: vi.fn(), removeEventListener: vi.fn() };
	const doc = {
		defaultView: scope,
		hidden: false,
		addEventListener: vi.fn(),
		removeEventListener: vi.fn()
	};
	const target = { postMessage: vi.fn() };
	const frame = {
		src: embed + '?context=unique',
		contentWindow: target,
		ownerDocument: doc,
		addEventListener: vi.fn(),
		removeEventListener: vi.fn()
	} as unknown as HTMLIFrameElement;
	const ready = vi.fn(),
		failed = vi.fn();
	const bridge = connectBunnyPlayer(frame, { ready, failed });
	const message = scope.addEventListener.mock.calls[0][1];
	const data = JSON.stringify({
		context: 'player.js',
		event: 'ready',
		value: { src: frame.src }
	});
	message({ origin: 'https://foreign.test', source: target, data });
	message({ origin: 'https://player.mediadelivery.net', source: {}, data });
	message({
		origin: 'https://player.mediadelivery.net',
		source: target,
		data: JSON.stringify({
			context: 'player.js',
			event: 'ready',
			value: { src: embed }
		})
	});
	expect(ready).not.toHaveBeenCalled();
	const load = (frame.addEventListener as ReturnType<typeof vi.fn>).mock
		.calls[0][1];
	load();
	message({ origin: 'https://player.mediadelivery.net', source: target, data });
	expect(ready).toHaveBeenCalledTimes(1);
	vi.advanceTimersByTime(20001);
	expect(failed).not.toHaveBeenCalled();
	observe([{ isIntersecting: false }]);
	expect(JSON.parse(target.postMessage.mock.calls.at(-1)![0]).method).toBe(
		'pause'
	);
	expect(target.postMessage.mock.calls.at(-1)![1]).toBe(
		'https://player.mediadelivery.net'
	);
	doc.hidden = true;
	doc.addEventListener.mock.calls[0][1]();
	bridge.destroy();
	expect(disconnect).toHaveBeenCalled();
	expect(scope.removeEventListener).toHaveBeenCalledWith('message', message);
	expect(doc.removeEventListener).toHaveBeenCalledWith(
		'visibilitychange',
		expect.any(Function)
	);
	expect(frame.removeEventListener).toHaveBeenCalledWith('load', load);
	const unavailable = vi.fn();
	const waiting = connectBunnyPlayer(frame, {
		ready: vi.fn(),
		failed: unavailable
	});
	(frame.addEventListener as ReturnType<typeof vi.fn>).mock.calls.at(-1)![1]();
	vi.advanceTimersByTime(20001);
	expect(unavailable).toHaveBeenCalledTimes(1);
	waiting.destroy();
});
