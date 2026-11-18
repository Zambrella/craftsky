import { it, expect } from 'vitest';
import { CraftskyAPI } from '#lib/api/client.ts';
import { FieldValidationError } from '#lib/field-validation.ts';
import { allFolders, normalizeSavedFolderName } from './saved-model';

it('normalizes folder names and counts Unicode scalars rather than UTF-16 units', () => {
	expect(normalizeSavedFolderName('  Knitting  ')).toBe('Knitting');
	expect(normalizeSavedFolderName('🧶'.repeat(100))).toBe('🧶'.repeat(100));
	expect(() => normalizeSavedFolderName('🧶'.repeat(101))).toThrow(
		'100 characters'
	);
});

it.each(['  ', 'knitting/sewing', 'knitting\\sewing', 'a\u0000b', 'a\u0085b'])(
	'rejects invalid folder names before writing: %j',
	(value) => {
		expect(() => normalizeSavedFolderName(value)).toThrow(FieldValidationError);
	}
);
it('shows later folder pages and rejects an endlessly repeated opaque cursor', async () => {
	const requests: string[] = [];
	const api = new CraftskyAPI(async (input) => {
		const url = String(input);
		requests.push(url);
		return Response.json(
			url.includes('cursor=')
				? { items: [{ id: 'two', name: 'Later folder' }] }
				: {
						items: [{ id: 'one', name: 'First folder' }],
						cursor: 'opaque/next?'
					}
		);
	});
	expect((await allFolders(api)).map((value) => value.name)).toEqual([
		'First folder',
		'Later folder'
	]);
	expect(requests[1]).toContain('cursor=opaque%2Fnext%3F');
	const bad = new CraftskyAPI(async () =>
		Response.json({ items: [], cursor: 'repeated' })
	);
	await expect(allFolders(bad)).rejects.toThrow('could not be completed');
});
