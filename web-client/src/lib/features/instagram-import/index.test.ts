import { afterEach, expect, it, vi } from 'vitest';
import { openInstagramArchive } from './index';

function stubWorker() {
	const worker = {
		postMessage: vi.fn(),
		terminate: vi.fn(),
		onmessage: (_event: { data: unknown }) => {}
	};
	vi.stubGlobal(
		'Worker',
		vi.fn(function () {
			return worker;
		})
	);
	return worker;
}

afterEach(() => vi.unstubAllGlobals());

it('terminates the archive worker when navigation interrupts opening', async () => {
	const worker = stubWorker();
	const controller = new AbortController();
	const opening = openInstagramArchive(
		new File([], 'export.zip'),
		controller.signal
	);
	controller.abort();
	await expect(opening).rejects.toMatchObject({ name: 'AbortError' });
	expect(worker.terminate).toHaveBeenCalledOnce();
});

it('transfers the opened worker to the retained archive session', async () => {
	const worker = stubWorker();
	const controller = new AbortController();
	const opening = openInstagramArchive(
		new File([], 'export.zip'),
		controller.signal
	);
	const id = worker.postMessage.mock.calls[0][0].id;
	worker.onmessage({ data: { id, value: { posts: [], skipped: 0 } } });
	const archive = await opening;
	controller.abort();
	expect(worker.terminate).not.toHaveBeenCalled();
	archive.close();
	expect(worker.terminate).toHaveBeenCalledOnce();
});
