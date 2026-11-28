import { InputError } from '#lib/user-facing-error.ts';
import { parseInstagramExport } from './archive';

self.onmessage = async ({ data }: MessageEvent<File>) => {
	try {
		self.postMessage({ result: await parseInstagramExport(data) });
	} catch (error) {
		// Only the parser's reviewed validation messages cross the worker boundary.
		self.postMessage({
			error: error instanceof InputError ? error.message : undefined
		});
	}
};
