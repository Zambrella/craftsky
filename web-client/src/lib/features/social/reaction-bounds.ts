export type ReactionRect = {
	left: number;
	top: number;
	right: number;
	bottom: number;
};
/** The feedback stays inside the initiating card's visible intersection. */
export function reactionBounds(card: ReactionRect, clips: ReactionRect[]) {
	let { left, top, right, bottom } = card;
	for (const clip of clips) {
		left = Math.max(left, clip.left);
		top = Math.max(top, clip.top);
		right = Math.min(right, clip.right);
		bottom = Math.min(bottom, clip.bottom);
	}
	const size = Math.min(160, right - left, bottom - top);
	return size > 0
		? { left: (left + right - size) / 2, top: (top + bottom - size) / 2, size }
		: undefined;
}
