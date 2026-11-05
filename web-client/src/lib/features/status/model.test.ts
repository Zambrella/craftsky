import { describe, it, expect } from 'vitest';
import { parseStatus, compare, readStatus } from './model';
const normal = { schemaVersion: 2, revision: 'revision-2' };
describe('independent service status', () => {
	it('validates schema, versions, safe URLs, independent controls', () => {
		expect(parseStatus(normal)).toEqual(normal);
		expect(compare('1.10.0', '1.9.8')).toBe(1);
		expect(compare('1.0.0-alpha.9', '1.0.0-alpha.10')).toBe(-1);
		expect(compare('1.0.0', '1.0.0-rc.1')).toBe(1);
		expect(compare('1.0.0+99', '1.0.0+1')).toBe(0);
		for (const extra of [
			{ schemaVersion: 1 },
			{ revision: 'x/../../' },
			{ maintenance: { title: '', description: 'x' } },
			{ minimumVersions: { web: 'x' } },
			{ minimumVersions: { web: '01.0.0' } },
			{
				maintenance: {
					title: 'x',
					description: 'x',
					estimatedRecoveryAt: '2026-02-30T12:00:00Z'
				}
			},
			{
				message: {
					revision: 'm',
					title: 'x',
					description: 'x',
					minVersion: '1.0.0',
					imageUrl: 'javascript:alert(1)'
				}
			},
			{
				message: {
					revision: 'm',
					title: 'x',
					description: 'x',
					minVersion: '1.0.0',
					cta: { text: 'Go', destination: '__proto__' }
				}
			}
		])
			expect(() => parseStatus({ ...normal, ...extra })).toThrow();
	});
	it('rejects streamed oversized bytes, invalid UTF8, wrong MIME and redirects', async () => {
		await expect(
			readStatus(
				new Response('x'.repeat(16385), {
					headers: { 'content-type': 'application/json' }
				})
			)
		).rejects.toThrow();
		await expect(
			readStatus(
				new Response(new Uint8Array([0xff]), {
					headers: { 'content-type': 'application/json' }
				})
			)
		).rejects.toThrow();
		await expect(
			readStatus(new Response('{}', { status: 302 }))
		).rejects.toThrow();
		await expect(
			readStatus(
				new Response(JSON.stringify(normal), {
					headers: { 'content-type': 'application/json; charset=utf-8' }
				})
			)
		).resolves.toEqual(normal);
	});
});
