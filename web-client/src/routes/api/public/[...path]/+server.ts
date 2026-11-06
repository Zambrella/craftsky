import { publicReadResponse } from '#lib/server/public-proxy.ts';
import type { RequestHandler } from './$types';
export const GET: RequestHandler = ({ fetch, params, url }) =>
	publicReadResponse(fetch, params.path, url.searchParams);
