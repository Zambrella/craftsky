import './context-menu.css';

const scrollLocks = new WeakMap<
	Document,
	{ previous: string; owners: Set<HTMLDetailsElement> }
>();

/** Responsive native menu presentation, dismissal, and compact keyboard focus. */
export function contextMenu(node: HTMLDetailsElement) {
	const document = node.ownerDocument;
	const view = document.defaultView!;
	node.classList.add('native-context-menu');
	const compact = view.matchMedia('(max-width: 900px)');
	const menu = node.querySelector<HTMLElement>('.menu-items');
	const summary = node.querySelector<HTMLElement>('summary');
	let locked = false;
	let wasOpen = false;
	let backdrop = node.querySelector<HTMLButtonElement>(
		'.menu-backdrop,.link-menu-backdrop'
	);
	const addedBackdrop = !backdrop;
	if (!backdrop) {
		backdrop = document.createElement('button');
		backdrop.type = 'button';
		backdrop.className = 'menu-backdrop';
		backdrop.setAttribute('aria-label', 'Close menu');
		node.insertBefore(backdrop, menu);
	}
	function lock() {
		if (locked) return;
		let state = scrollLocks.get(document);
		if (!state) {
			state = {
				previous: document.documentElement.style.overflow,
				owners: new Set()
			};
			scrollLocks.set(document, state);
		}
		state.owners.add(node);
		document.documentElement.style.overflow = 'hidden';
		locked = true;
	}
	function unlock() {
		if (!locked) return;
		const state = scrollLocks.get(document);
		state?.owners.delete(node);
		if (state && state.owners.size === 0) {
			document.documentElement.style.overflow = state.previous;
			scrollLocks.delete(document);
		}
		locked = false;
	}
	function outside(event: Event) {
		if (node.open && !node.contains(event.target as Node)) node.open = false;
	}
	function escape(event: KeyboardEvent) {
		if (!node.open) return;
		if (event.key === 'Tab' && compact.matches) {
			const controls = [
				...(menu?.querySelectorAll<HTMLElement>(
					'a[href],button:not([disabled]),input:not([disabled])'
				) || [])
			].filter((item) => item.getClientRects().length);
			const first = controls[0],
				last = controls.at(-1);
			if (
				event.shiftKey &&
				(document.activeElement === first ||
					!menu?.contains(document.activeElement))
			) {
				event.preventDefault();
				last?.focus();
			} else if (
				!event.shiftKey &&
				(document.activeElement === last ||
					!menu?.contains(document.activeElement))
			) {
				event.preventDefault();
				first?.focus();
			}
			return;
		}
		if (event.key !== 'Escape') return;
		event.preventDefault();
		node.open = false;
		node.querySelector<HTMLElement>('summary')?.focus();
	}
	function choose(event: MouseEvent) {
		const target = (event.target as Element | null)?.closest?.('button,a');
		if (target && !target.hasAttribute('disabled')) {
			node.open = false;
			unlock();
		}
	}
	function position() {
		if (!node.open) return;
		if (!menu || !summary) return;
		if (compact.matches) {
			lock();
			return;
		}
		unlock();
		const rect = summary.getBoundingClientRect();
		const height = view.innerHeight;
		const upward = rect.top > height / 2;
		menu.style.top = upward ? 'auto' : `${rect.bottom + 4}px`;
		menu.style.bottom = upward ? `${height - rect.top + 4}px` : 'auto';
		menu.style.maxHeight = `${Math.max(100, (upward ? rect.top : height - rect.bottom) - 12)}px`;
		menu.style.left = `${Math.max(8, Math.min(rect.right - menu.offsetWidth, view.innerWidth - menu.offsetWidth - 8))}px`;
		menu.style.right = 'auto';
		menu.style.overflowY = 'auto';
	}
	function toggle() {
		if (node.open) {
			for (const other of document.querySelectorAll<HTMLDetailsElement>(
				'.native-context-menu[open]'
			))
				if (other !== node) other.open = false;
			position();
			if (!wasOpen && compact.matches)
				menu
					?.querySelector<HTMLElement>(
						'a[href],button:not([disabled]),input:not([disabled])'
					)
					?.focus();
		} else {
			unlock();
			if (
				menu?.contains(document.activeElement) ||
				document.activeElement === backdrop
			)
				summary?.focus();
		}
		wasOpen = node.open;
	}
	document.addEventListener('pointerdown', outside);
	document.addEventListener('keydown', escape);
	node.addEventListener('click', choose, true);
	node.addEventListener('toggle', toggle);
	view.addEventListener('resize', position);
	view.addEventListener('scroll', position, true);
	return {
		destroy() {
			document.removeEventListener('pointerdown', outside);
			document.removeEventListener('keydown', escape);
			node.removeEventListener('click', choose, true);
			node.removeEventListener('toggle', toggle);
			view.removeEventListener('resize', position);
			view.removeEventListener('scroll', position, true);
			unlock();
			if (addedBackdrop) backdrop?.remove();
		}
	};
}
