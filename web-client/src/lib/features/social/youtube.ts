export interface YouTubeExternal {
	videoId: string;
	isShort: boolean;
	startSeconds: number;
}

// Match the mobile parser. Never use post content as an iframe URL.
export function parseYouTubeExternal(
	value: string
): YouTubeExternal | undefined {
	let url: URL;
	try {
		url = new URL(value);
	} catch {
		return;
	}
	if (!['http:', 'https:'].includes(url.protocol)) return;
	let parts: string[];
	try {
		parts = url.pathname.split('/').filter(Boolean).map(decodeURIComponent);
	} catch {
		return;
	}
	let videoId: string | null | undefined;
	let isShort = false;
	if (url.hostname === 'youtu.be') videoId = parts[0];
	else if (
		[
			'youtube.com',
			'www.youtube.com',
			'm.youtube.com',
			'music.youtube.com'
		].includes(url.hostname)
	) {
		if (parts[0] === 'shorts' || parts[0] === 'live') {
			videoId = parts[1];
			isShort = parts[0] === 'shorts';
		} else if (!parts.length || parts[0] === 'watch')
			videoId = url.searchParams.get('v');
	}
	if (!videoId || !/^[A-Za-z0-9_-]{11}$/.test(videoId)) return;
	const timestamp =
		url.searchParams.get('t') ?? url.searchParams.get('start') ?? '';
	let startSeconds = 0;
	if (/^[+-]?\d+$/.test(timestamp))
		startSeconds = Math.max(0, Number(timestamp));
	else {
		const match = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/.exec(timestamp);
		if (match)
			startSeconds =
				Number(match[1] || 0) * 3600 +
				Number(match[2] || 0) * 60 +
				Number(match[3] || 0);
	}
	return {
		videoId,
		isShort,
		startSeconds: Number.isSafeInteger(startSeconds) ? startSeconds : 0
	};
}

type Player = { destroy(): void };
type YouTubeAPI = {
	Player: new (
		element: HTMLIFrameElement,
		options: { events: { onError: () => void } }
	) => Player;
};
let apiPromise: Promise<YouTubeAPI> | undefined;

// Called only after the viewer has consented to contacting YouTube.
export function loadYouTubeAPI(): Promise<YouTubeAPI> {
	if (apiPromise) return apiPromise;
	apiPromise = new Promise((resolve, reject) => {
		const scope = window as typeof window & {
			YT?: YouTubeAPI;
			onYouTubeIframeAPIReady?: () => void;
		};
		if (scope.YT?.Player) {
			resolve(scope.YT);
			return;
		}
		const script = document.createElement('script');
		script.src = 'https://www.youtube.com/iframe_api';
		const timeout = setTimeout(fail, 15000);
		function fail() {
			clearTimeout(timeout);
			script.remove();
			apiPromise = undefined;
			reject(new Error('YouTube player unavailable'));
		}
		script.onerror = fail;
		scope.onYouTubeIframeAPIReady = () => {
			clearTimeout(timeout);
			if (scope.YT?.Player) resolve(scope.YT);
			else fail();
		};
		document.head.append(script);
	});
	return apiPromise;
}
