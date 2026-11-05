import wasmURL from '@rive-app/canvas/rive.wasm?url';
export type RiveAnimation = {
	runtime: typeof import('@rive-app/canvas');
	buffer: ArrayBuffer;
};
let runtime: Promise<RiveAnimation['runtime']> | undefined;
const animations = new Map<string, Promise<RiveAnimation>>();
/** Shared public animation bytes/runtime; each playback owns its controller. */
export function loadRiveAnimation(url: string): Promise<RiveAnimation> {
	const cached = animations.get(url);
	if (cached) return cached;
	runtime ||= import('@rive-app/canvas')
		.then(async (module) => {
			module.RuntimeLoader.setWasmUrl(wasmURL);
			module.RuntimeLoader.setWasmFallbackUrl(null);
			await module.RuntimeLoader.awaitInstance();
			return module;
		})
		.catch((error) => {
			runtime = undefined;
			throw error;
		});
	const loading = Promise.all([
		runtime,
		fetch(url).then((response) => {
			if (!response.ok) throw new Error('Animation unavailable');
			return response.arrayBuffer();
		})
	])
		.then(([runtime, buffer]) => ({ runtime, buffer }))
		.catch((error) => {
			animations.delete(url);
			throw error;
		});
	animations.set(url, loading);
	return loading;
}
