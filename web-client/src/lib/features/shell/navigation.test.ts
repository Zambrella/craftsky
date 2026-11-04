import { expect, it } from 'vitest';
import { ownProfileRoute } from './navigation';
const account = { did: 'did:plc:alice', handle: 'alice.test' };
it('recognizes DID, percent-encoded DID and current handle routes', () => {
	for (const address of [
		'did:plc:alice',
		'did%3Aplc%3Aalice',
		'did%3aplc%3aalice',
		'alice.test',
		'ALICE.TEST',
		'renamed.test'
	])
		expect(
			ownProfileRoute('/app/profile/' + address, account, 'renamed.test')
		).toBe(true);
});
it('excludes other profiles, editing, nested paths and malformed encoding', () => {
	for (const path of [
		'/app/profile/did%3Aplc%3Abob',
		'/app/profile/edit',
		'/app/profile/alice.test/posts',
		'/app/home',
		'/app/profile/%ZZ'
	])
		expect(ownProfileRoute(path, account)).toBe(false);
});
