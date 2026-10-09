const MAX_BYTES = 16384;
const HEADERS = {
  'Content-Type': 'application/json; charset=utf-8',
  'Cache-Control': 'no-store',
  'Access-Control-Allow-Origin': '*',
  'X-Content-Type-Options': 'nosniff',
};
function unavailable(status, head = false, extra = {}) {
  return new Response(head ? null : '{"error":"statusUnavailable"}', {status, headers: {...HEADERS, ...extra}});
}
export default {
  async fetch(request, env) {
    const head = request.method === 'HEAD';
    if (new URL(request.url).pathname !== '/app.json') return unavailable(404, head);
    if (request.method === 'OPTIONS') {
      return new Response(null, {status: 204, headers: {...HEADERS,
        'Access-Control-Allow-Methods': 'GET, HEAD, OPTIONS', 'Access-Control-Allow-Headers': 'Accept'}});
    }
    if (!['GET', 'HEAD'].includes(request.method)) return unavailable(405, head, {Allow: 'GET, HEAD, OPTIONS'});
    try {
      // Binding reads bypass public R2 caching. There is no Cache API or write path.
      const object = await env.STATUS_DOCUMENTS.get('app.json');
      if (!object) return unavailable(404, head);
      if (!Number.isInteger(object.size) || object.size < 0 || object.size > MAX_BYTES) return unavailable(503, head);
      if (head) return new Response(null, {headers: HEADERS});
      const bytes = await object.arrayBuffer();
      if (bytes.byteLength > MAX_BYTES) return unavailable(503);
      return new Response(bytes, {headers: HEADERS});
    } catch {
      console.warn(JSON.stringify({feature: 'ServiceStatus', operation: 'read', outcome: 'unavailable'}));
      return unavailable(503, head);
    }
  },
};
