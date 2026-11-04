interface ReorderOptions {
	group: string;
	index: number;
	enabled?: boolean;
	onmove: (from: number, to: number) => void;
}

/** Lift the entire row; keep the original space and animate its neighbours. */
export function reorderRow(node: HTMLElement, options: ReorderOptions) {
	let cleanup: (() => void) | undefined;
	function down(event: PointerEvent) {
		const handle = (event.target as Element).closest<HTMLElement>(
			'.drag-handle'
		);
		if (!handle || options.enabled === false || event.button !== 0) return;
		event.preventDefault();
		const from = options.index;
		const rows = [
			...node.parentElement!.querySelectorAll<HTMLElement>(
				':scope > [data-reorder-group]'
			)
		].filter(
			(row) =>
				row.dataset.reorderGroup === options.group &&
				row.dataset.reorderEnabled !== 'false'
		);
		const boxes = rows.map((row) => row.getBoundingClientRect());
		const box = node.getBoundingClientRect();
		let scroller: HTMLElement | null = node.parentElement;
		while (
			scroller &&
			!(
				scroller.scrollHeight > scroller.clientHeight &&
				/auto|scroll/.test(getComputedStyle(scroller).overflowY)
			)
		)
			scroller = scroller.parentElement;
		const scrollTop = () => (scroller ? scroller.scrollTop : window.scrollY);
		const initialScroll = scrollTop();
		let y = event.clientY,
			to = from,
			ghost: HTMLElement | undefined,
			frame = 0;
		const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
		handle.setPointerCapture(event.pointerId);
		function paint() {
			if (!ghost) return;
			const offset = scrollTop() - initialScroll;
			ghost.style.top = `${box.top + y - event.clientY}px`;
			const centre = box.top + box.height / 2 + y - event.clientY + offset;
			to = from;
			boxes.forEach((other, i) => {
				if (i < from && centre < other.top + other.height / 2)
					to = Math.min(to, i);
				if (i > from && centre > other.top + other.height / 2)
					to = Math.max(to, i);
			});
			const gap =
				boxes.length > 1 ? Math.max(0, boxes[1].top - boxes[0].bottom) : 0;
			rows.forEach((row, i) => {
				if (row === node) return;
				const direction =
					i > from && i <= to ? -1 : i < from && i >= to ? 1 : 0;
				row.style.transition = reduced ? 'none' : 'transform 120ms ease-out';
				row.style.transform = `translateY(${direction * (box.height + gap)}px)`;
			});
		}
		function scroll() {
			if (!ghost) return;
			const bounds = scroller?.getBoundingClientRect();
			const top = bounds?.top ?? 0,
				bottom = bounds?.bottom ?? innerHeight;
			const speed = y < top + 48 ? -10 : y > bottom - 48 ? 10 : 0;
			if (speed) {
				(scroller || window).scrollBy(0, speed);
				paint();
			}
			frame = requestAnimationFrame(scroll);
		}
		function move(e: PointerEvent) {
			if (e.pointerId !== event.pointerId) return;
			y = e.clientY;
			if (!ghost && Math.abs(y - event.clientY) < 4) return;
			if (!ghost) {
				ghost = node.cloneNode(true) as HTMLElement;
				ghost.removeAttribute('data-reorder-group');
				ghost.setAttribute('aria-hidden', 'true');
				ghost.inert = true;
				ghost.classList.add('reorder-ghost');
				Object.assign(ghost.style, {
					position: 'fixed',
					left: `${box.left}px`,
					top: `${box.top}px`,
					width: `${box.width}px`,
					height: `${box.height}px`,
					margin: '0',
					zIndex: '110',
					pointerEvents: 'none',
					background: 'var(--surface)',
					boxShadow: '0 8px 24px #0004',
					transform: 'scale(1.015)',
					borderRadius: '12px'
				});
				node.parentElement!.append(ghost);
				node.style.opacity = '0.15';
				document.documentElement.style.cursor = 'grabbing';
				frame = requestAnimationFrame(scroll);
			}
			paint();
		}
		function finish(e: PointerEvent) {
			if (e.pointerId !== event.pointerId) return;
			const moved = !!ghost && e.type === 'pointerup';
			cleanup?.();
			if (moved && from !== to) options.onmove(from, to);
		}
		function escape(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				e.preventDefault();
				cleanup?.();
			}
		}
		cleanup = () => {
			cancelAnimationFrame(frame);
			ghost?.remove();
			rows.forEach((row) => {
				row.style.removeProperty('transform');
				row.style.removeProperty('transition');
			});
			node.style.removeProperty('opacity');
			document.documentElement.style.removeProperty('cursor');
			if (handle.hasPointerCapture(event.pointerId))
				handle.releasePointerCapture(event.pointerId);
			document.removeEventListener('pointermove', move);
			document.removeEventListener('pointerup', finish);
			document.removeEventListener('pointercancel', finish);
			document.removeEventListener('keydown', escape);
			cleanup = undefined;
		};
		document.addEventListener('pointermove', move);
		document.addEventListener('pointerup', finish);
		document.addEventListener('pointercancel', finish);
		document.addEventListener('keydown', escape);
	}
	function update(next: ReorderOptions) {
		options = next;
		node.dataset.reorderGroup = options.group;
		node.dataset.reorderEnabled = String(options.enabled !== false);
	}
	update(options);
	node.addEventListener('pointerdown', down);
	return {
		update,
		destroy() {
			cleanup?.();
			node.removeEventListener('pointerdown', down);
		}
	};
}
