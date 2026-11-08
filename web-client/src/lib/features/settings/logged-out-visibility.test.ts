import { describe, it, expect } from 'vitest';
import { LoggedOutVisibility } from './logged-out-visibility.svelte';
import { MutationOutcomeUnknownError } from '#lib/api/transport.ts';
describe('UT-006 shared visibility setting', () => {
	it('has no false public default and keeps the original pending command until confirmed', async () => {
		const calls: unknown[] = [];
		let attempts = 0;
		const model = new LoggedOutVisibility(
			{
				read: async () => ({ hideFromLoggedOut: false, expectedCid: 'old' }),
				save: async (body, key) => {
					calls.push({ body, key });
					if (++attempts === 1)
						throw new MutationOutcomeUnknownError(key, 1, true);
					return { hideFromLoggedOut: true, expectedCid: 'new' };
				}
			},
			() => '11111111-1111-4111-8111-111111111111'
		);
		expect(model.value).toBeUndefined();
		await model.load();
		await model.setHidden(true);
		expect(model.value?.hideFromLoggedOut).toBe(false);
		expect(model.pending).toBeDefined();
		await model.retry();
		expect(calls[0]).toEqual(calls[1]);
		expect(model.pending).toBeUndefined();
		expect(model.value?.hideFromLoggedOut).toBe(true);
	});
	it('recovers pending intent on owner reload and ignores completion after disposal', async () => {
		const pending = {
			operationKey: '11111111-1111-4111-8111-111111111111',
			expectedCid: 'old',
			hideFromLoggedOut: true
		};
		const calls: unknown[] = [];
		const model = new LoggedOutVisibility({
			read: async () => ({
				hideFromLoggedOut: true,
				expectedCid: 'observed',
				pending
			}),
			save: async (body, key) => {
				calls.push({ body, key });
				return { hideFromLoggedOut: true, expectedCid: 'new' };
			}
		});
		await model.load();
		await model.retry();
		expect(calls).toEqual([
			{
				body: { hideFromLoggedOut: true, expectedCid: 'old' },
				key: pending.operationKey
			}
		]);
		const late = new LoggedOutVisibility({
			read: async () => ({ hideFromLoggedOut: false, expectedCid: 'old' }),
			save: async () => ({ hideFromLoggedOut: true, expectedCid: 'new' })
		});
		const loading = late.load();
		late.dispose();
		await loading;
		expect(late.value).toBeUndefined();
	});
	it('does not call an unresolved acknowledgement saved', async () => {
		const intent = {
			hideFromLoggedOut: true,
			expectedCid: 'old',
			operationKey: 'original-key'
		};
		const model = new LoggedOutVisibility({
			read: async () => ({
				hideFromLoggedOut: false,
				expectedCid: 'old',
				pending: intent
			}),
			save: async () => ({
				hideFromLoggedOut: false,
				expectedCid: 'old',
				pending: intent
			})
		});
		await model.load();
		await model.retry();
		expect(model.pending).toEqual(intent);
		expect(model.notice).toBe('');
		expect(model.value?.hideFromLoggedOut).toBe(false);
	});
});
