import { describe, it, expect } from 'vitest';
import { authenticatedContext, routeTitle } from './page-context';
describe('shared page context', () => {
	it.each([true, false])(
		'reads the same release configuration on private and public pages (%s)',
		(enabled) => {
			const data = {
				session: { id: 'account', did: 'did:plc:a', handle: 'a.test' },
				accounts: [],
				subscriptionsEnabled: enabled
			};
			expect(authenticatedContext(data)).toEqual(data);
			expect(authenticatedContext({ authenticated: data })).toEqual(data);
			expect(authenticatedContext({})).toBeUndefined();
		}
	);
	it('gives routes useful titles without including identifiers or private queries', () => {
		expect(routeTitle('/feed')).toBe('Feed · CraftSky');
		expect(routeTitle('/settings/privacy')).toBe('Privacy settings · CraftSky');
		expect(routeTitle('/messages/conversations/PRIVATE_RECIPIENT')).toBe(
			'Conversation · CraftSky'
		);
		expect(routeTitle('/compose/edit')).toBe('Edit post · CraftSky');
	});
});
