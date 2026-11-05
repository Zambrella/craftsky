import { facetSegments } from './facet-input';

/** Keep native textarea editing/IME/selection, painting facets in a noninteractive mirror. */
export function textareaPresentation(
	node: HTMLTextAreaElement,
	options: {
		value: string;
		position: (point: { left: number; top: number }) => void;
	}
) {
	const mirror = document.createElement('div');
	const measure = document.createElement('div');
	for (const layer of [mirror, measure]) {
		layer.setAttribute('aria-hidden', 'true');
		layer.style.cssText =
			'position:absolute;pointer-events:none;overflow:hidden;margin:0;';
		node.before(layer);
	}
	mirror.className = 'facet-text-paint';
	measure.style.visibility = 'hidden';
	function sync() {
		const style = getComputedStyle(node);
		for (const layer of [mirror, measure]) {
			for (const property of [
				'font-family',
				'font-size',
				'font-weight',
				'font-style',
				'line-height',
				'letter-spacing',
				'text-align',
				'text-indent',
				'text-transform',
				'tab-size',
				'padding-top',
				'padding-right',
				'padding-bottom',
				'padding-left',
				'border-top-width',
				'border-right-width',
				'border-bottom-width',
				'border-left-width',
				'border-radius',
				'box-sizing'
			]) {
				layer.style.setProperty(property, style.getPropertyValue(property));
			}
			layer.style.borderStyle = 'solid';
			layer.style.borderColor = 'transparent';
			layer.style.whiteSpace = node.wrap === 'off' ? 'pre' : 'pre-wrap';
			layer.style.overflowWrap = 'break-word';
			layer.style.top = node.offsetTop + 'px';
			layer.style.left = node.offsetLeft + 'px';
			layer.style.width = node.offsetWidth + 'px';
			layer.style.height = node.offsetHeight + 'px';
		}
		mirror.replaceChildren(
			...facetSegments(options.value).map((part) => {
				const span = document.createElement('span');
				span.textContent = part.text;
				span.style.color = part.facet ? 'var(--primary)' : 'var(--ink)';
				return span;
			})
		);
		mirror.append(document.createTextNode('\u200b'));
		mirror.scrollTop = node.scrollTop;
		mirror.scrollLeft = node.scrollLeft;
		const caret = document.createElement('span');
		caret.textContent = '\u200b';
		measure.replaceChildren(
			document.createTextNode(node.value.slice(0, node.selectionStart)),
			caret,
			document.createTextNode(node.value.slice(node.selectionStart) || '\u200b')
		);
		measure.scrollTop = node.scrollTop;
		measure.scrollLeft = node.scrollLeft;
		const rect = caret.getBoundingClientRect();
		const parent = node.parentElement!.getBoundingClientRect();
		const field = node.getBoundingClientRect();
		options.position({
			left: Math.max(
				0,
				Math.min(
					rect.left - parent.left,
					parent.width - Math.min(320, parent.width)
				)
			),
			top:
				Math.max(
					field.top - parent.top + 8,
					Math.min(field.bottom - parent.top, rect.bottom - parent.top)
				) + 4
		});
	}
	const events = ['input', 'scroll', 'click', 'keyup', 'select'];
	for (const event of events) node.addEventListener(event, sync);
	const observer = new ResizeObserver(sync);
	observer.observe(node);
	sync();
	return {
		update(next: typeof options) {
			options = next;
			sync();
		},
		destroy() {
			observer.disconnect();
			for (const event of events) node.removeEventListener(event, sync);
			mirror.remove();
			measure.remove();
		}
	};
}
