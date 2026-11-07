import type { Video } from '#lib/api/client.ts';
const origin = 'https://player.mediadelivery.net';

/** Only AppView's explicit Bunny destination selects the hosted player. */
export function bunnyEmbedURL(video: Video): string | undefined {
	if (!video.embedUrl || video.captions?.length) return;
	try {
		const url = new URL(video.embedUrl);
		if (
			url.origin !== origin ||
			url.username ||
			url.password ||
			url.search ||
			url.hash ||
			!/^\/embed\/[1-9]\d*\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(
				url.pathname
			)
		)
			return;
		return url.href;
	} catch {
		return;
	}
}

/** A scoped subset of Bunny's player.js protocol; all listeners are disposable. */
export function connectBunnyPlayer(
	frame: HTMLIFrameElement,
	callbacks: { ready: () => void; failed: () => void }
) {
	const scope = frame.ownerDocument.defaultView!;
	const listener = crypto.randomUUID();
	let ready = false;
	let timer: ReturnType<typeof setTimeout> | undefined;
	const send = (method: string, value?: string) =>
		frame.contentWindow?.postMessage(
			JSON.stringify({
				context: 'player.js',
				version: '0.0.11',
				method,
				value,
				listener
			}),
			origin
		);
	const pause = () => {
		if (ready) send('pause');
	};
	const message = (event: MessageEvent) => {
		if (event.origin !== origin || event.source !== frame.contentWindow) return;
		let data;
		try {
			if (typeof event.data !== 'string' || event.data.length > 32768) return;
			data = JSON.parse(event.data);
		} catch {
			return;
		}
		if (data?.context !== 'player.js') return;
		if (data.event === 'ready' && data.value?.src === frame.src && !ready) {
			ready = true;
			clearTimeout(timer);
			callbacks.ready();
			send('addEventListener', 'error');
			if (frame.ownerDocument.hidden) pause();
		} else if (data.event === 'error') callbacks.failed();
	};
	const loaded = () => {
		if (ready) return;
		send('addEventListener', 'ready');
		clearTimeout(timer);
		timer = setTimeout(callbacks.failed, 20000);
	};
	const visibility = () => {
		if (frame.ownerDocument.hidden) pause();
	};
	const observer = new IntersectionObserver((entries) => {
		if (entries.some((entry) => !entry.isIntersecting)) pause();
	});
	observer.observe(frame);
	scope.addEventListener('message', message);
	frame.addEventListener('load', loaded);
	frame.ownerDocument.addEventListener('visibilitychange', visibility);
	return {
		destroy() {
			pause();
			clearTimeout(timer);
			observer.disconnect();
			scope.removeEventListener('message', message);
			frame.removeEventListener('load', loaded);
			frame.ownerDocument.removeEventListener('visibilitychange', visibility);
		}
	};
}
