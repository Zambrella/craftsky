import { it, expect } from 'vitest';
import { allowedRoute } from './routes';
it('IT-007 exposes only authenticated GET and PUT for the shared preference', () => {
	for (const method of ['GET', 'PUT'])
		expect(allowedRoute(method, '/v1/preferences/logged-out-visibility')).toBe(
			true
		);
	for (const method of ['POST', 'DELETE', 'PATCH'])
		expect(allowedRoute(method, '/v1/preferences/logged-out-visibility')).toBe(
			false
		);
	expect(
		allowedRoute('PUT', '/v1/public/preferences/logged-out-visibility')
	).toBe(false);
});
