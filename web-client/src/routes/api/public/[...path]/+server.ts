import { publicReadResponse } from '#lib/server/public-proxy.ts';
import { publicReportResponse } from '#lib/server/public-report-proxy.ts';
import type { RequestHandler } from './$types';
export const GET: RequestHandler = ({ fetch, params, url }) =>
	publicReadResponse(fetch, params.path, url.searchParams);
export const POST: RequestHandler = ({ request, params, url }) =>
	publicReportResponse(request, url, params.path);
