import adapter from '@sveltejs/adapter-cloudflare';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter(),
			csp: {
				mode: 'hash',
				directives: {
					'default-src': ['self'],
					'script-src': ['self'],
					'style-src': ['self', 'unsafe-inline'],
					'img-src': ['self', 'https:'],
					'media-src': ['https:'],
					'connect-src': ['self'],
					'base-uri': ['none'],
					'form-action': ['none'],
					'frame-ancestors': ['none'],
					'object-src': ['none']
				}
			}
		})
	],
	test: { include: ['src/**/*.test.ts'] }
});
