import { defineConfig } from '@playwright/test';
import { fileURLToPath } from 'node:url';
const cwd = fileURLToPath(new URL('..', import.meta.url));
const subscriptions = process.env.WEB_CLIENT_SUBSCRIPTIONS_ENABLED || 'true';
export default defineConfig({
	testDir: '.',
	outputDir: '../test-results-authenticated',
	testMatch: 'authenticated-*.spec.ts',
	fullyParallel: false,
	workers: 1,
	use: {
		baseURL: 'http://127.0.0.1:19846',
		javaScriptEnabled: true,
		viewport: { width: 1280, height: 900 }
	},
	webServer: [
		{
			command: 'node tests/authenticated-appview.mjs',
			cwd,
			port: 19847,
			reuseExistingServer: false
		},
		{
			command:
				process.env.WEB_CLIENT_WORKER === '1'
					? `npx wrangler pages dev .svelte-kit/cloudflare --ip 127.0.0.1 --port 19846 --binding APPVIEW_ORIGIN=http://127.0.0.1:19847 --binding SUBSCRIPTIONS_ENABLED=${subscriptions}`
					: 'npm run preview -- --port 19846',
			cwd,
			port: 19846,
			env: {
				APPVIEW_ORIGIN: 'http://127.0.0.1:19847',
				SUBSCRIPTIONS_ENABLED: subscriptions
			},
			reuseExistingServer: false
		}
	]
});
