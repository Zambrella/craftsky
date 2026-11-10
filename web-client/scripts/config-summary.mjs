import { loadEnv } from 'vite';
import { pathToFileURL } from 'node:url';

export function summarize(env) {
	const present = (key) => Boolean(env[key]?.trim());
	const push = [
		'PUBLIC_FIREBASE_API_KEY',
		'PUBLIC_FIREBASE_APP_ID',
		'PUBLIC_FIREBASE_PROJECT_ID',
		'PUBLIC_FIREBASE_MESSAGING_SENDER_ID',
		'PUBLIC_FIREBASE_VAPID_KEY'
	].every(present);
	return [
		'Web services and feature flags (configuration; runtime consent still applies):',
		`  Firebase browser push: ${push ? 'configured (requires permission)' : 'disabled (configuration incomplete)'}`,
		`  Sentry: ${present('PUBLIC_SENTRY_DSN') ? 'configured (requires consent)' : 'disabled'}`,
		`  SUBSCRIPTIONS_ENABLED: ${env.SUBSCRIPTIONS_ENABLED === 'true' ? 'enabled' : 'disabled'}`,
		'Backend-controlled features are reported by just dev-d; connectivity is not checked.'
	].join('\n');
}

export function launchSummary(args, directory) {
	let mode = 'development';
	for (let i = 0; i < args.length; i++) {
		if (args[i] === '--mode' || args[i] === '-m') mode = args[++i];
		else if (args[i].startsWith('--mode=')) mode = args[i].slice(7);
	}
	// Vite owns dotenv expansion, file precedence and shell overrides.
	return summarize(loadEnv(mode, directory, ''));
}

if (
	process.argv[1] &&
	import.meta.url === pathToFileURL(process.argv[1]).href
) {
	try {
		console.log(launchSummary(process.argv.slice(2), process.cwd()));
	} catch {
		console.error('Web configuration summary unavailable; values withheld.');
	}
}
