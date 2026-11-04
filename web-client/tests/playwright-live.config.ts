import { defineConfig } from '@playwright/test';
if (process.env.E2E_LIVE !== '1') throw new Error('Use npm run test:e2e:live.');
const baseURL = process.env.E2E_BASE_URL!;
const port = new URL(baseURL).port;
export default defineConfig({
	testDir: './live',
	testMatch: 'smoke.spec.ts',
	workers: 1,
	retries: 0,
	timeout: 300_000,
	expect: { timeout: 15_000 },
	outputDir: process.env.E2E_OUTPUT,
	preserveOutput: 'never',
	reporter: [['./live/reporter.ts']],
	use: {
		baseURL,
		browserName: 'chromium',
		viewport: { width: 1280, height: 900 },
		trace: 'off',
		screenshot: 'off',
		video: 'off',
		actionTimeout: 15_000,
		navigationTimeout: 60_000
	},
	webServer: {
		command: `npm run dev -- --port ${port} --strictPort`,
		url: baseURL + '/sign-in',
		reuseExistingServer: false,
		timeout: 90_000,
		env: {
			APPVIEW_ORIGIN: process.env.APPVIEW_ORIGIN!,
			E2E_HANDLE: '',
			E2E_PASSWORD: '',
			PUBLIC_SENTRY_DSN: ''
		},
		stdout: 'ignore',
		stderr: 'ignore'
	}
});
