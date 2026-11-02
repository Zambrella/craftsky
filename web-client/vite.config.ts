import adapter from '@sveltejs/adapter-cloudflare';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import { readFileSync } from 'node:fs';
export default defineConfig({
	define: {
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
			serviceWorker: { register: false },
			csp: {
				mode: 'hash',
				directives: {
					'default-src': ['self'],
					'script-src': ['self'],
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
		})
	],
	test: { include: ['src/**/*.test.ts'] }
});
