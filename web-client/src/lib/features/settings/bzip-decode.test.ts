import { describe, expect, it } from 'vitest';
import { decodeBzip2 } from './bzip-decode.ts';
import { bzipLimits } from './bzip-limits.ts';
import fixture from './bzip-fixture.json';

const compressed = () =>
	Uint8Array.from(atob(fixture.bzip), (byte) => byte.charCodeAt(0));
describe('bounded BZip2 byte-stream decoder', () => {
	it('decodes the real BZip2 fixture without allocating an output larger than the declaration', () => {
		expect(
			new TextDecoder().decode(decodeBzip2(compressed(), fixture.json.length))
		).toBe(fixture.json);
		expect(() => decodeBzip2(compressed(), 1)).toThrow('declared size');
		expect(() => decodeBzip2(compressed(), bzipLimits.outputBytes + 1)).toThrow(
			'output size'
		);
	});
	it('verifies the BZip2 block CRC and requires the complete footer without trailing bytes', () => {
		const damaged = compressed();
		damaged[10] ^= 1;
		expect(() => decodeBzip2(damaged, fixture.json.length)).toThrow();
		expect(() => decodeBzip2(compressed().subarray(0, 4), 0)).toThrow(
			'Truncated'
		);
		expect(() =>
			decodeBzip2(compressed().subarray(0, -1), fixture.json.length)
		).toThrow();
		expect(() =>
			decodeBzip2(new Uint8Array([...compressed(), 0]), fixture.json.length)
		).toThrow('size');
	});
	it('checks its CPU deadline independently of the declared archive size', () => {
		expect(() => decodeBzip2(compressed(), fixture.json.length, 0)).toThrow(
			'timed out'
		);
	});
});
