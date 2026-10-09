import test from 'node:test';
import assert from 'node:assert/strict';
import worker from '../src/index.js';
const raw = JSON.stringify({schemaVersion: 1, mode: 'maintenance', revision: 'A', title: 'Public', message: 'Notice'});
function store(value = raw) {
  const keys = [];
  return {keys, binding: {async get(key) {
    keys.push(key);
    return {size: Buffer.byteLength(value), async arrayBuffer() {return new TextEncoder().encode(value).buffer;}};
  }, put() {throw new Error('public write forbidden');}}};
}
function headers(response) {
  assert.equal(response.headers.get('content-type'), 'application/json; charset=utf-8');
  assert.equal(response.headers.get('cache-control'), 'no-store');
  assert.equal(response.headers.get('access-control-allow-origin'), '*');
  assert.equal(response.headers.get('access-control-allow-credentials'), null);
  assert.equal(response.headers.get('set-cookie'), null);
  assert.equal(response.headers.get('x-content-type-options'), 'nosniff');
}
// IT-007 / FR-001, NFR-002, RULE-002, RULE-003 / AC-001, AC-016, AC-019.
test('GET and HEAD use fixed private binding anonymously with uncached headers', async () => {
  const fake = store();
  for (const method of ['GET', 'HEAD']) {
    const response = await worker.fetch(new Request('https://status.test/app.json', {method}), {STATUS_DOCUMENTS: fake.binding});
    assert.equal(response.status, 200); headers(response);
    assert.equal(await response.text(), method === 'HEAD' ? '' : raw);
  }
  assert.deepEqual(fake.keys, ['app.json', 'app.json']);
});
test('every request observes the current object and code reload leaves it alone', async () => {
  let current = raw; let writes = 0;
  const binding = {get: async () => ({size: Buffer.byteLength(current), arrayBuffer: async () => new TextEncoder().encode(current).buffer}),
    put: () => {writes++;}};
  const read = async (handler) => (await handler.fetch(new Request('https://status.test/app.json'), {STATUS_DOCUMENTS: binding})).json();
  assert.equal((await read(worker)).revision, 'A');
  current = JSON.stringify({schemaVersion: 1, mode: 'normal', revision: 'B'});
  assert.equal((await read(worker)).revision, 'B');
  const reloaded = (await import('../src/index.js?code-reload')).default;
  assert.equal((await read(reloaded)).revision, 'B'); assert.equal(writes, 0);
});
test('CORS preflight is read-only and does not touch storage', async () => {
  const response = await worker.fetch(new Request('https://status.test/app.json', {method: 'OPTIONS',
    headers: {Origin: 'https://app.test', 'Access-Control-Request-Method': 'GET'}}), {});
  assert.equal(response.status, 204);
  assert.equal(response.headers.get('access-control-allow-origin'), '*');
  assert.equal(response.headers.get('access-control-allow-methods'), 'GET, HEAD, OPTIONS');
  assert.equal(response.headers.get('cache-control'), 'no-store');
});
test('other paths/mutations never touch the binding and have bounded errors', async () => {
  const fake = store();
  for (const [path, method, status] of [['/other', 'GET', 404], ['/app.json', 'POST', 405], ['/app.json', 'PUT', 405], ['/app.json', 'DELETE', 405]]) {
    const response = await worker.fetch(new Request(`https://status.test${path}`, {method}), {STATUS_DOCUMENTS: fake.binding});
    assert.equal(response.status, status); headers(response);
    assert.ok((await response.text()).length < 128);
  }
  assert.deepEqual(fake.keys, []);
});
test('missing, oversized and failed objects stay unknown without diagnostic prose', async () => {
  const lines = []; const original = console.warn; console.warn = (...args) => lines.push(args.join(' '));
  try {
    for (const binding of [{get: async () => null}, store('x'.repeat(16385)).binding,
      {get: async () => {throw new Error('credential-and-prose-canary');}}]) {
      const response = await worker.fetch(new Request('https://status.test/app.json'), {STATUS_DOCUMENTS: binding});
      assert.ok(response.status >= 400); headers(response);
      assert.ok((await response.text()).length < 128);
    }
  } finally {console.warn = original;}
  assert.match(lines.join('\n'), /ServiceStatus/);
  assert.doesNotMatch(lines.join('\n'), /credential-and-prose-canary/);
});
