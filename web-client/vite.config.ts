import adapter from '@sveltejs/adapter-cloudflare';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import { readFileSync } from 'node:fs';
import { sentryVitePlugin } from '@sentry/vite-plugin';
import { webRelease } from './scripts/sentry-build.mjs';
const sentryBuildPlugin = () =>
	sentryVitePlugin({
		telemetry: false,
		sourcemaps: { disable: 'disable-upload' },
		// Only debug-ID injection is needed here; runtime release metadata is explicit.
		release: {
			name: '',
			inject: false,
			create: false,
			finalize: false,
			setCommits: false,
			deploy: false
		}
	});
export default defineConfig({
	build: { sourcemap: 'hidden' },
	worker: { plugins: () => [sentryBuildPlugin()] },
	define: {
		__CRAFTSKY_WEB_RELEASE__: JSON.stringify(webRelease),
		__CRAFTSKY_VERSION__: JSON.stringify(
			readFileSync(
				new URL('../appview/VERSION', import.meta.url),
				'utf8'
			).trim()
		)
	},
	plugins: [
		sveltekit({
			adapter: adapter(),
			files: { params: 'src/params.ts' },
			serviceWorker: { register: false },
			csp: {
				mode: 'hash',
				directives: {
					'default-src': ['self'],
					'script-src': ['self', 'wasm-unsafe-eval'],
					'style-src': ['self', 'unsafe-inline'],
					'img-src': ['self', 'https:', 'blob:', 'data:'],
					'media-src': ['self', 'https:', 'blob:'],
					'connect-src': ['self', 'https:'],
					'base-uri': ['none'],
					'form-action': ['self'],
					'frame-ancestors': ['none'],
					'object-src': ['none']
				}
			}
		}),
		sentryBuildPlugin()
	],
	test: { include: ['src/**/*.test.ts'] }
});
