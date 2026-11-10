import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { launchSummary, summarize } from './config-summary.mjs';

test('reports incomplete push, consent and exact subscription flag without values', () => {
	const text = summarize({
		PUBLIC_SENTRY_DSN: 'secret-canary',
		SUBSCRIPTIONS_ENABLED: 'TRUE'
	});
	assert.match(text, /Firebase browser push: disabled/);
	assert.match(text, /Sentry: configured \(requires consent\)/);
	assert.match(text, /SUBSCRIPTIONS_ENABLED: disabled/);
	assert.ok(!text.includes('secret-canary'));
	assert.match(
		summarize(
			Object.fromEntries(
				[
					'API_KEY',
					'APP_ID',
					'PROJECT_ID',
					'MESSAGING_SENDER_ID',
					'VAPID_KEY'
				].map((key) => [`PUBLIC_FIREBASE_${key}`, 'canary'])
			)
		),
		/Firebase browser push: configured/
	);
});

test('uses Vite mode-specific dotenv precedence and shell overrides', () => {
	const directory = mkdtempSync(join(tmpdir(), 'craftsky-summary-'));
	mkdirSync(join(directory, 'config'));
	const previous = process.env.SUBSCRIPTIONS_ENABLED;
	try {
		delete process.env.SUBSCRIPTIONS_ENABLED;
		writeFileSync(
			join(directory, 'config/.env.production.local'),
			'SUBSCRIPTIONS_ENABLED=false\n'
		);
		writeFileSync(
			join(directory, 'config/.env.development.local'),
			'SUBSCRIPTIONS_ENABLED=true\n'
		);
		writeFileSync(
			join(directory, 'config/.env.review.local'),
			'SUBSCRIPTIONS_ENABLED=false\n'
		);
		assert.match(
			launchSummary([], directory),
			/SUBSCRIPTIONS_ENABLED: enabled/
		);
		assert.match(
			launchSummary(['--mode', 'review'], directory),
			/SUBSCRIPTIONS_ENABLED: disabled/
		);
		process.env.SUBSCRIPTIONS_ENABLED = 'true';
		assert.match(
			launchSummary(['--mode=review'], directory),
			/SUBSCRIPTIONS_ENABLED: enabled/
		);
	} finally {
		if (previous === undefined) delete process.env.SUBSCRIPTIONS_ENABLED;
		else process.env.SUBSCRIPTIONS_ENABLED = previous;
		rmSync(directory, { recursive: true });
	}
});

test('development public settings never enter production and generic files are rejected', () => {
	const directory = mkdtempSync(join(tmpdir(), 'craftsky-modes-'));
	mkdirSync(join(directory, 'config'));
	const previous = process.env.PUBLIC_SENTRY_DSN;
	try {
		delete process.env.PUBLIC_SENTRY_DSN;
		writeFileSync(
			join(directory, 'config/.env.development.local'),
			'PUBLIC_SENTRY_DSN=dev-canary\n'
		);
		assert.match(launchSummary([], directory), /Sentry: configured/);
		assert.match(
			launchSummary(['--mode=production'], directory),
			/Sentry: disabled/
		);
		writeFileSync(
			join(directory, 'config/.env.production.local'),
			'PUBLIC_SENTRY_DSN=prod-canary\n'
		);
		assert.match(
			launchSummary(['--mode=production'], directory),
			/Sentry: configured/
		);
		writeFileSync(
			join(directory, 'config/.env.local'),
			'PUBLIC_SENTRY_DSN=ambiguous-canary\n'
		);
		assert.throws(
			() => launchSummary(['--mode=production'], directory),
			/Use config\/\.env\.development\.local/
		);
	} finally {
		if (previous === undefined) delete process.env.PUBLIC_SENTRY_DSN;
		else process.env.PUBLIC_SENTRY_DSN = previous;
		rmSync(directory, { recursive: true });
	}
});
