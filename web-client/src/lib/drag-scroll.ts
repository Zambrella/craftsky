/** Mouse dragging scrolls a tab strip without accidentally activating a tab. */
export function dragScroll(node: HTMLElement) {
	let gesture:
		{ id: number; x: number; scroll: number; moved: boolean } | undefined;
	let suppressClick = false;
	function down(event: PointerEvent) {
		if (
			event.pointerType !== 'mouse' ||
			event.button !== 0 ||
			node.scrollWidth <= node.clientWidth
		)
			return;
		suppressClick = false;
		gesture = {
			id: event.pointerId,
			x: event.clientX,
			scroll: node.scrollLeft,
			moved: false
		};
	}
	function move(event: PointerEvent) {
		if (!gesture || event.pointerId !== gesture.id) return;
		const delta = event.clientX - gesture.x;
		if (!gesture.moved && Math.abs(delta) < 5) return;
		gesture.moved = true;
		node.setPointerCapture(event.pointerId);
		node.dataset.dragging = '';
		node.scrollLeft = gesture.scroll - delta;
		event.preventDefault();
	}
	function end() {
		if (!gesture) return;
		suppressClick = gesture.moved;
		if (node.hasPointerCapture(gesture.id))
			node.releasePointerCapture(gesture.id);
		gesture = undefined;
		delete node.dataset.dragging;
	}
	function click(event: MouseEvent) {
		if (!suppressClick) return;
		suppressClick = false;
		event.preventDefault();
		event.stopImmediatePropagation();
	}
	function drag(event: DragEvent) {
		event.preventDefault();
	}
	node.addEventListener('pointerdown', down);
	window.addEventListener('pointermove', move, { passive: false });
	window.addEventListener('pointerup', end);
	window.addEventListener('pointercancel', end);
	node.addEventListener('click', click, true);
	node.addEventListener('dragstart', drag);
	return {
		destroy() {
			end();
			node.removeEventListener('pointerdown', down);
			window.removeEventListener('pointermove', move);
			window.removeEventListener('pointerup', end);
			window.removeEventListener('pointercancel', end);
			node.removeEventListener('click', click, true);
			node.removeEventListener('dragstart', drag);
		}
	};
}
