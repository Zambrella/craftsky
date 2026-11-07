import { spawnSync } from 'node:child_process';
// Diagnostic fixture builds are temporary; a successful gate restores the caller's build environment.
const fixture = {
	...process.env,
	PUBLIC_SENTRY_DSN: 'http://public@127.0.0.1:19846/1',
	PUBLIC_SENTRY_ENVIRONMENT: 'test',
	WEB_CLIENT_DIAGNOSTICS: '1',
	WEB_CLIENT_SUBSCRIPTIONS_ENABLED: 'true'
};
const run = (script, env = process.env, args = []) => {
	const result = spawnSync('npm', ['run', script, ...args], {
		stdio: 'inherit',
		env
	});
	if (result.error) throw result.error;
	if (result.status !== 0) process.exit(result.status ?? 1);
};
for (const script of ['format:check', 'lint', 'check', 'test']) run(script);
run('build', fixture);
for (const worker of ['0', '1']) {
	const env = { ...fixture, WEB_CLIENT_WORKER: worker };
	run('test:e2e', env);
	run('test:e2e:app', env);
	run(
		'test:e2e:app',
		{
			...env,
			WEB_CLIENT_SUBSCRIPTIONS_ENABLED: 'false'
		},
		[
			'--',
			'authenticated-beta-release.spec.ts',
			'authenticated-ipad-parity.spec.ts',
			'--grep',
			'beta release hides|native release hides'
		]
	);
}

// The deployment script uploads the artifact left by this gate. Never leave a
// loopback diagnostic build as the successful output. Failure stops before upload.
run('build');
