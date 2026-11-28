export const postLimits = {
	text: { graphemes: 2000, bytes: 20000 },
	imageCount: 8,
	imageSourceBytes: 40_000_000,
	videoSourceBytes: 300_000_000
} as const;

export function videoSourceMime(
	header: Uint8Array
): 'video/mp4' | 'video/quicktime' | undefined {
	if (header.length < 12) return;
	const tag = new TextDecoder('ascii').decode(header.subarray(4, 12));
	if (tag === 'ftypqt  ') return 'video/quicktime';
	if (
		['ftypisom', 'ftypiso2', 'ftypmp41', 'ftypmp42', 'ftypavc1'].includes(tag)
	)
		return 'video/mp4';
}
