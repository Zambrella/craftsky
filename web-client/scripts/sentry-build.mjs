import { readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
const root = new URL('../../', import.meta.url);
const version = readFileSync(
	new URL('web-client/VERSION', root),
	'utf8'
).trim();
const commit = execFileSync('git', ['rev-parse', '--short=12', 'HEAD'], {
	cwd: root,
	encoding: 'utf8'
}).trim();
export const webRelease =
	process.env.SENTRY_RELEASE || `craftsky-web@${version}+${commit}`;
if (!/^craftsky-web@[0-9A-Za-z.+-]{1,100}$/.test(webRelease))
	throw new Error(
		'SENTRY_RELEASE must start with craftsky-web@ and contain only version/build characters'
	);
