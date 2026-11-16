export function contentBounds(node: HTMLElement) {
	const update = () => {
		const box = node.getBoundingClientRect();
		document.documentElement.style.setProperty(
			'--app-content-left',
			`${box.left}px`
		);
		document.documentElement.style.setProperty(
			'--app-content-width',
			`${box.width}px`
		);
	};
	const observer = new ResizeObserver(update);
	observer.observe(node);
	// The discovery column can move a capped-width main column without resizing it.
	const layout = new MutationObserver(update);
	layout.observe(node, {
		attributes: true,
		attributeFilter: ['class', 'style']
	});
	let frame: number | undefined;
	const resized = () => {
		if (frame !== undefined) cancelAnimationFrame(frame);
		// Read after responsive shell state and styles have settled.
		frame = requestAnimationFrame(() => {
			frame = undefined;
			update();
		});
	};
	window.addEventListener('resize', resized);
	update();
	return {
		destroy() {
			observer.disconnect();
			layout.disconnect();
			window.removeEventListener('resize', resized);
			if (frame !== undefined) cancelAnimationFrame(frame);
		}
	};
}
