import { defineConfig } from '@playwright/test';
export default defineConfig({
	testDir: './tests',
	testMatch: 'public-pages.spec.ts',
	use: { baseURL: 'http://127.0.0.1:19840', javaScriptEnabled: false },
	webServer: [
		{
			command: 'node tests/mock-appview.mjs',
			port: 19841,
			reuseExistingServer: false
		},
		{
			command:
				process.env.WEB_CLIENT_WORKER === '1'
					? 'npx wrangler pages dev .svelte-kit/cloudflare --ip 127.0.0.1 --port 19840 --binding APPVIEW_ORIGIN=http://127.0.0.1:19841'
					: 'npm run preview -- --port 19840',
			port: 19840,
			env: { APPVIEW_ORIGIN: 'http://127.0.0.1:19841' },
			reuseExistingServer: false
		}
	]
});
