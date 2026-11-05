import { defineParams } from '@sveltejs/kit/params';
import { authenticatedRoute } from '#lib/routes.ts';

// Keep app navigation from claiming server-only OAuth/API URLs.
export const params = defineParams({
	authenticated: (path: string) =>
		authenticatedRoute('/' + path) ? path : undefined
});
