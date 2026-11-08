import { describe, expect, it } from 'vitest';
import { parseYouTubeExternal } from './youtube';

describe('mobile YouTube URL parity', () => {
	it.each([
		['https://www.youtube.com/watch?v=dQw4w9WgXcQ', false, 0],
		['https://youtu.be/dQw4w9WgXcQ?t=90', false, 90],
		['https://m.youtube.com/shorts/dQw4w9WgXcQ', true, 0],
		['https://youtube.com/live/dQw4w9WgXcQ?start=20', false, 20],
		['https://music.youtube.com/watch?v=dQw4w9WgXcQ&t=1h2m3s', false, 3723],
		['https://youtube.com/watch?v=dQw4w9WgXcQ&t=-2', false, 0],
		['https://youtu.be/dQw4w9WgXcQ?t=bad&start=20', false, 0]
	])('recognizes %s', (url, isShort, startSeconds) => {
		expect(parseYouTubeExternal(url as string)).toEqual({
			videoId: 'dQw4w9WgXcQ',
			isShort,
			startSeconds
		});
	});
	it.each([
		'https://youtube.com.example.org/watch?v=dQw4w9WgXcQ',
		'https://notyoutube.com/watch?v=dQw4w9WgXcQ',
		'https://youtube.com/embed/dQw4w9WgXcQ',
		'https://youtube.com/watch?v=too-short',
		'https://youtu.be/%ZZ',
		'javascript:alert(1)',
		'not a url'
	])('keeps unsupported URLs as links: %s', (url) => {
		expect(parseYouTubeExternal(url)).toBeUndefined();
	});
});
