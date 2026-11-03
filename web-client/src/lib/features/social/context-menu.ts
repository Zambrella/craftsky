/** Native-like dismissal for semantic details menus, without consuming their actions. */
export function contextMenu(node: HTMLDetailsElement) {
	const document = node.ownerDocument;
	function outside(event: Event) {
		if (node.open && !node.contains(event.target as Node)) node.open = false;
	}
	function escape(event: KeyboardEvent) {
		if (!node.open || event.key !== 'Escape') return;
		event.preventDefault();
		node.open = false;
		node.querySelector<HTMLElement>('summary')?.focus();
	}
	function choose(event: MouseEvent) {
		const target = (event.target as Element | null)?.closest?.('button,a');
		if (target && !target.hasAttribute('disabled')) node.open = false;
	}
	function position() {
		if (!node.open) return;
		const menu = node.querySelector<HTMLElement>('.menu-items');
		const summary = node.querySelector<HTMLElement>('summary');
		if (!menu || !summary) return;
		const rect = summary.getBoundingClientRect();
		const height = document.defaultView?.innerHeight || 800;
		const upward = rect.top > height / 2;
		menu.style.top = upward ? 'auto' : '100%';
		menu.style.bottom = upward ? '100%' : 'auto';
		menu.style.maxHeight = `${Math.max(100, (upward ? rect.top : height - rect.bottom) - 16)}px`;
		menu.style.overflowY = 'auto';
	}
	document.addEventListener('pointerdown', outside);
	document.addEventListener('keydown', escape);
	node.addEventListener('click', choose);
	node.addEventListener('toggle', position);
	return {
		destroy() {
			document.removeEventListener('pointerdown', outside);
			document.removeEventListener('keydown', escape);
			node.removeEventListener('click', choose);
			node.removeEventListener('toggle', position);
		}
	};
}
