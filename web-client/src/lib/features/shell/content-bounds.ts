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
	window.addEventListener('resize', update);
	update();
	return {
		destroy() {
			observer.disconnect();
			window.removeEventListener('resize', update);
		}
	};
}
