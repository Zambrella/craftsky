import js from '@eslint/js';
import ts from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';
import prettier from 'eslint-config-prettier';
import globals from 'globals';
export default ts.config(
	{
		ignores: [
			'.deployments/**',
			'.wrangler/**',
			'.svelte-kit/**',
			'.sentry-artifacts/**',
			'build/**',
			'test-results*/**',
			'playwright-report/**',
			'node_modules/**',
			'static/**'
		]
	},
	js.configs.recommended,
	...ts.configs.recommended,
	...svelte.configs['flat/recommended'],
	prettier,
	{
		files: ['tests/**/*.mjs'],
		rules: { '@typescript-eslint/ban-ts-comment': 'off' }
	},
	{ files: ['tests/live/*.ts'], rules: { 'no-empty-pattern': 'off' } },
	...svelte.configs['flat/prettier'],
	{
		languageOptions: { globals: { ...globals.browser, ...globals.node } },
		rules: {
			// Intentional best-effort cleanup and protected fallbacks may have empty catches.
			'no-empty': ['error', { allowEmptyCatch: true }],
			'@typescript-eslint/no-unused-vars': [
				'error',
				{
					argsIgnorePattern: '^_',
					caughtErrors: 'none',
					varsIgnorePattern: '^_'
				}
			],
			'@typescript-eslint/no-explicit-any': 'off',
			// The app is deployed at the origin root; URL helpers already validate external links.
			'svelte/no-navigation-without-resolve': 'off',
			// Existing static lists and intentionally non-reactive collections are valid.
			'svelte/require-each-key': 'off',
			'svelte/no-useless-mustaches': 'off',
			'svelte/prefer-svelte-reactivity': 'off',
			'no-control-regex': 'off',
			'no-useless-escape': ['error', { allowRegexCharacters: ['-'] }]
		}
	},
	{
		files: ['**/*.svelte', '**/*.svelte.ts'],
		rules: {
			'@typescript-eslint/no-unused-expressions': 'off',
			'no-undef': 'off'
		},
		languageOptions: { parserOptions: { parser: ts.parser } }
	}
);
