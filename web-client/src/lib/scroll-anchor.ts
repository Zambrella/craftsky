/** Capture surviving rows, in reading order, immediately before a DOM update. */
export function captureScrollAnchor(container: HTMLElement): () => void {
	if (window.scrollY <= 0) return () => {};
	const candidates = [
		...container.querySelectorAll<HTMLElement>('[data-scroll-anchor]')
	]
		.map((node) => ({
			key: node.dataset.scrollAnchor,
			top: node.getBoundingClientRect().top,
			bottom: node.getBoundingClientRect().bottom
		}))
		.filter((row) => row.bottom > 0);
	return () => {
		if (!container.isConnected) return;
		const rows = new Map(
			[...container.querySelectorAll<HTMLElement>('[data-scroll-anchor]')].map(
				(node) => [node.dataset.scrollAnchor, node]
			)
		);
		for (const candidate of candidates) {
			const node = rows.get(candidate.key);
			if (!node) continue;
			const delta = node.getBoundingClientRect().top - candidate.top;
			if (Math.abs(delta) > 0.5)
				window.scrollBy({ top: delta, behavior: 'instant' });
			return;
		}
	};
}
