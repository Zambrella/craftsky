export function mountComposerPage(node: HTMLElement) {
	// Retain editor ancestry and scoped field styles while showing its detail page.
	const scroller = node.closest<HTMLElement>('.composer-scroll');
	const scrollTop = scroller?.scrollTop || 0;
	const hidden: { node: HTMLElement; display: string; inert: boolean }[] = [];
	let branch: HTMLElement = node;
	while (branch.parentElement && branch.parentElement.tagName !== 'MAIN') {
		for (const sibling of branch.parentElement.children) {
			if (
				sibling === branch ||
				!(sibling instanceof HTMLElement) ||
				sibling.tagName === 'DIALOG'
			)
				continue;
			hidden.push({
				node: sibling,
				display: sibling.style.display,
				inert: sibling.inert
			});
			sibling.style.display = 'none';
			sibling.inert = true;
		}
		branch = branch.parentElement;
	}
	node.querySelector<HTMLElement>('button')?.focus({ preventScroll: true });
	return {
		destroy() {
			for (const item of hidden) {
				item.node.style.display = item.display;
				item.node.inert = item.inert;
			}
			if (scroller) scroller.scrollTop = scrollTop;
		}
	};
}
