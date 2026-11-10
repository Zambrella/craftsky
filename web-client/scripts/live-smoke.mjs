// Dedicated-account live runner. Never targets a remote AppView or resets other accounts.
import { readFileSync, mkdtempSync, rmSync } from 'node:fs';
import { parseEnv } from 'node:util';
import { execFileSync, spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const root = fileURLToPath(new URL('../../', import.meta.url));
const cwd = join(root, 'web-client');
const args = process.argv.slice(2);
if (args.some((arg) => !['--list', '--headed'].includes(arg))) {
	throw new Error('Supported live smoke options: --list, --headed');
}
const listing = args.includes('--list');
let values;
try {
	const source = readFileSync(join(cwd, 'config/.env.local'), 'utf8');
	values = parseEnv(source);
	// Match Patrol's password handling: an unquoted # is part of the password.
	// Quoted values still follow dotenv escaping rules. Never accept inline comments here.
	const rawPassword = source
		.split(/\r?\n/)
		.find((line) => /^E2E_PASSWORD=/.test(line))
		?.slice('E2E_PASSWORD='.length)
		.trim();
	if (rawPassword && !/^["']/.test(rawPassword))
		values.E2E_PASSWORD = rawPassword;
} catch {
	throw new Error(
		'Add E2E_HANDLE and E2E_PASSWORD to web-client/config/.env.local.'
	);
}
if (!values.E2E_HANDLE || !values.E2E_PASSWORD)
	throw new Error(
		'Set E2E_HANDLE and E2E_PASSWORD in web-client/config/.env.local.'
	);
const compose = (...command) =>
	execFileSync(join(root, 'scripts/compose-dev'), command, {
		cwd: root,
		encoding: 'utf8',
		stdio: ['pipe', 'pipe', 'pipe'],
		timeout: 30_000
	}).trim();
let appview = 'http://127.0.0.1:8080',
	base = 'http://127.0.0.1:19848',
	did = '';
try {
	if (!listing) {
		const address = compose('port', 'appview', '8080');
		if (!/^127\.0\.0\.1:\d+$/.test(address)) throw new Error();
		appview = 'http://' + address;
		base = compose('exec', '-T', 'appview', 'printenv', 'WEB_APP_ORIGIN');
		if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(base)) throw new Error();
		if (
			!(
				await fetch(appview + '/healthz', {
					signal: AbortSignal.timeout(15_000)
				})
			).ok
		)
			throw new Error();
		const result = await fetch(
			'https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?' +
				new URLSearchParams({ handle: values.E2E_HANDLE }),
			{ signal: AbortSignal.timeout(20_000) }
		);
		if (!result.ok) throw new Error();
		did = (await result.json()).did;
		if (!/^did:plc:[a-z2-7]{24}$/.test(did)) throw new Error();
		// Same reset as scripts/app-test-e2e, restricted to the discovered local database.
		execFileSync(
			join(root, 'scripts/compose-dev'),
			[
				'exec',
				'-T',
				'postgres',
				'psql',
				'-X',
				'-v',
				'ON_ERROR_STOP=1',
				'-U',
				'craftsky',
				'-d',
				'craftsky_dev',
				'-v',
				`e2e_did=${did}`
			],
			{
				cwd: root,
				input:
					"DELETE FROM account_onboarding_completions WHERE account_did = :'e2e_did';\n",
				stdio: ['pipe', 'pipe', 'pipe'],
				timeout: 30_000
			}
		);
	}
} catch {
	throw new Error(
		'Live smoke preparation failed. Start this worktree with just dev-d; its WEB_APP_ORIGIN must be an explicit 127.0.0.1 HTTP origin and the dedicated Bluesky account must resolve.'
	);
}
const output = mkdtempSync(join(tmpdir(), 'craftsky-web-live-'));
const environment = {
	...process.env,
	E2E_HANDLE: values.E2E_HANDLE,
	E2E_PASSWORD: values.E2E_PASSWORD,
	E2E_DID: did,
	E2E_LIVE: '1',
	E2E_BASE_URL: base,
	E2E_OUTPUT: output,
	APPVIEW_ORIGIN: appview,
	DEBUG: '',
	PWDEBUG: '',
	PUBLIC_SENTRY_DSN: '',
	PUBLIC_SENTRY_ENVIRONMENT: 'test'
};
console.log(
	listing
		? 'Listing live smoke test (no account changes).'
		: 'Running live smoke against this worktree. The dedicated account bio and public test post will be changed.'
);
const child = spawn(
	process.execPath,
	[
		'node_modules/@playwright/test/cli.js',
		'test',
		'--config',
		'tests/playwright-live.config.ts',
		...args
	],
	{ cwd, env: environment, stdio: ['ignore', 'pipe', 'pipe'] }
);
const protectedValues = [
	values.E2E_HANDLE,
	values.E2E_PASSWORD,
	encodeURIComponent(values.E2E_PASSWORD),
	JSON.stringify(values.E2E_PASSWORD).slice(1, -1)
]
	.filter(Boolean)
	.sort((a, b) => b.length - a.length);
// Buffer whole lines so a secret split between stream chunks is still redacted.
for (const stream of [child.stdout, child.stderr]) {
	let pending = '';
	stream.setEncoding('utf8');
	const print = (line) => {
		for (const value of protectedValues)
			line = line.replaceAll(value, '[redacted]');
		process.stdout.write(line);
	};
	stream.on('data', (chunk) => {
		pending += chunk;
		let end;
		while ((end = pending.indexOf('\n')) !== -1) {
			print(pending.slice(0, end + 1));
			pending = pending.slice(end + 1);
		}
	});
	stream.on('end', () => print(pending));
}
process.on('SIGINT', () => child.kill('SIGINT'));
process.on('SIGTERM', () => child.kill('SIGTERM'));
child.on('close', (code) => {
	rmSync(output, { recursive: true, force: true });
	process.exitCode = code ?? 1;
});
