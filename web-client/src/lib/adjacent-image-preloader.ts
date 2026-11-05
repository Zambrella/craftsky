export type AdjacentImages = {
	urls: (string | undefined)[];
	index: number;
	active?: boolean;
	onDecoded?: () => void;
};
/** Warm the displayed URL's immediate neighbours after its successful decode. */
export function preloadAdjacent(
	node: HTMLImageElement,
	initial: AdjacentImages
) {
	let options = initial;
	let signature = '';
	let revision = 0;
	let stop = () => {};
	let neighbours: HTMLImageElement[] = [];
	function update(next: AdjacentImages) {
		options = next;
		const key = JSON.stringify([next.urls, next.index, next.active !== false]);
		if (key === signature) return;
		signature = key;
		stop();
		neighbours = [];
		const currentRevision = ++revision;
		const currentURL = next.urls[next.index];
		if (next.active === false || !currentURL || next.urls.length < 2) return;
		let started = false;
		const loaded = () => {
			if (
				started ||
				!node.complete ||
				!node.naturalWidth ||
				node.src !== currentURL
			)
				return;
			started = true;
			void node
				.decode()
				.then(() => {
					if (currentRevision !== revision || node.src !== currentURL) return;
					const urls = new Set([
						next.urls[next.index - 1],
						next.urls[next.index + 1]
					]);
					for (const url of urls) {
						if (!url || url === currentURL) continue;
						const image = new window.Image();
						image.decoding = 'async';
						image.fetchPriority = 'low';
						image.src = url;
						neighbours.push(image);
						void image.decode().catch(() => {});
					}
					options.onDecoded?.();
				})
				.catch(() => {}); // Visible errors belong to the image; speculative failures stay silent.
		};
		node.addEventListener('load', loaded);
		stop = () => node.removeEventListener('load', loaded);
		loaded(); // A cached image may already have completed before the action attaches.
	}
	update(initial);
	return {
		update,
		destroy() {
			revision++;
			stop();
			neighbours = [];
		}
	};
}
