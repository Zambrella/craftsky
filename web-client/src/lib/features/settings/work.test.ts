import { describe, it, expect } from 'vitest';
import { MutationOutcomeUnknownError } from '#lib/api/client.ts';
import { Work } from './work.svelte.ts';

describe('settings async work', () => {
	it('does not start reads or writes after disposal', async () => {
		const work = new Work();
		work.dispose();
		let calls = 0;
		const request = async () => ++calls;
		await work.load(request, () => {});
		await work.run(request);
		expect(calls).toBe(0);
	});
	it('suppresses a late completion after disposal', async () => {
		const work = new Work();
		let finish!: (value: string) => void;
		let applied = '';
		const request = work.load(
			() =>
				new Promise<string>((resolve) => {
					finish = resolve;
				}),
			(value) => {
				applied = value;
			}
		);
		work.dispose();
		finish('previous account');
		await request;
		expect(applied).toBe('');
	});
	it('clears busy on errors and retains the frozen request for an ambiguous write', async () => {
		const work = new Work();
		let attempts = 0;
		const key = 'original-operation-key';
		const observed: string[] = [];
		const request = () => {
			observed.push(key);
			attempts++;
			return attempts === 1
				? Promise.reject(
						new MutationOutcomeUnknownError(
							'00000000-0000-4000-8000-000000000001'
						)
					)
				: Promise.resolve('done');
		};
		let result = '';
		await work.run(request, (value) => {
			result = value;
		});
		expect(work.busy).toBe(false);
		expect(work.retry).not.toBeNull();
		let unrelated = false;
		await work.run(async () => {
			unrelated = true;
		});
		expect(unrelated).toBe(false);
		await work.retry?.();
		expect(observed).toEqual([key, key]);
		expect(result).toBe('done');
		expect(work.retry).toBeNull();
		expect(work.busy).toBe(false);
	});
	it('prevents an older read replacing a newer load', async () => {
		const work = new Work();
		let first!: (value: string) => void;
		let displayed = '';
		const old = work.load(
			() =>
				new Promise<string>((resolve) => {
					first = resolve;
				}),
			(value) => (displayed = value)
		);
		await work.load(
			async () => 'current',
			(value) => (displayed = value)
		);
		first('stale');
		await old;
		expect(displayed).toBe('current');
		expect(work.loading).toBe(false);
	});
});
