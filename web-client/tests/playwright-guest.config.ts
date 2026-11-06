import { defineConfig, devices } from '@playwright/test';
export default defineConfig({
	outputDir: '../test-results-guest',
	workers: 1,
	testDir: '.',
	testMatch: 'guest-*.spec.ts',
	use: {
		baseURL: 'http://127.0.0.1:19850',
		javaScriptEnabled: true,
		trace: 'retain-on-failure'
	},
	projects: [
		{ name: 'chromium-desktop', use: { ...devices['Desktop Chrome'] } },
		{ name: 'chromium-mobile', use: { ...devices['Pixel 7'] } },
		{ name: 'webkit-mobile', use: { ...devices['iPhone 13'] } },
		{
			name: 'chromium-tablet',
			use: { ...devices['iPad (gen 7)'], defaultBrowserType: 'chromium' }
		}
	],
	webServer: [
		{
			cwd: new URL('..', import.meta.url).pathname,
			command: 'node tests/guest-appview.mjs',
			port: 19851,
			reuseExistingServer: false
		},
		{
			cwd: new URL('..', import.meta.url).pathname,
			command:
				process.env.WEB_CLIENT_WORKER === '1'
					? 'npx wrangler dev --ip 127.0.0.1 --port 19850 --var APPVIEW_ORIGIN:http://127.0.0.1:19851'
					: 'npm run preview -- --port 19850',
			port: 19850,
			env: { APPVIEW_ORIGIN: 'http://127.0.0.1:19851' },
			reuseExistingServer: false
		}
	]
});
