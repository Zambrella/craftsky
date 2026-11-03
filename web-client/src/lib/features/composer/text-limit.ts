const segments = new Intl.Segmenter(undefined, { granularity: 'grapheme' });
const encoder = new TextEncoder();
export type TextLimit = { graphemes: number; bytes: number };

/** Match Flutter's committed-edit limits without splitting emoji or combining marks. */
export function limitText(value: string, limit: TextLimit): string {
	let count = 0,
		bytes = 0,
		end = 0;
	for (const { segment, index } of segments.segment(value)) {
		bytes += encoder.encode(segment).length;
		if (++count > limit.graphemes || bytes > limit.bytes) break;
		end = index + segment.length;
	}
	return value.slice(0, end);
}

/** Existing values remain untouched until an edit; composition is committed as a unit. */
export function recordTextLimit(
	node: HTMLInputElement | HTMLTextAreaElement,
	limit: TextLimit | undefined
) {
	let composing = false;
	function apply(event: Event) {
		if (
			!limit ||
			composing ||
			(event instanceof InputEvent && event.isComposing)
		)
			return;
		const value = limitText(node.value, limit);
		if (value === node.value) return;
		const start = node.selectionStart ?? value.length;
		const end = node.selectionEnd ?? start;
		node.value = value;
		node.setSelectionRange(
			Math.min(start, value.length),
			Math.min(end, value.length)
		);
	}
	const begin = () => {
		composing = true;
	};
	const finish = (event: Event) => {
		composing = false;
		apply(event);
		node.dispatchEvent(new Event('input', { bubbles: true }));
	};
	node.addEventListener('input', apply, true);
	node.addEventListener('compositionstart', begin, true);
	node.addEventListener('compositionend', finish, true);
	return {
		update(next: TextLimit | undefined) {
			limit = next;
		},
		destroy() {
			node.removeEventListener('input', apply, true);
			node.removeEventListener('compositionstart', begin, true);
			node.removeEventListener('compositionend', finish, true);
		}
	};
}
