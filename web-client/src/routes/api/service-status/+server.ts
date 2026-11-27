import { readStatus } from '#lib/features/status/model.ts';

const headers = { 'cache-control': 'no-store' };

export async function GET() {
	try {
		const upstream = await fetch('https://status.craftsky.social/app.json', {
			credentials: 'omit',
			redirect: 'error',
			cache: 'no-store',
			headers: { Accept: 'application/json' },
			signal: AbortSignal.timeout(3000)
		});
		return Response.json(await readStatus(upstream), { headers });
	} catch {
		return new Response(null, { status: 503, headers });
	}
}
