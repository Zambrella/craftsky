import { readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
const root = new URL('../', import.meta.url);
const { release } = JSON.parse(
	readFileSync(new URL('.sentry-artifacts/release.json', root), 'utf8')
);
// Upload exactly the files retained by the completed build. Never rebuild or mutate them here.
execFileSync(
	'sentry',
	[
		'sourcemap',
		'upload',
		'.sentry-artifacts/_app',
		'--release',
		release,
		'--url-prefix',
		'~/_app/',
		'--no-rewrite'
	],
	{
		cwd: root,
		stdio: 'inherit',
		env: {
			...process.env,
			SENTRY_ORG: 'craftsky',
			SENTRY_PROJECT: 'web-client'
		}
	}
);
