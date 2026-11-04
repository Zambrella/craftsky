import { cp, mkdir, readdir, readFile, rm, writeFile } from 'node:fs/promises';
import { webRelease } from './sentry-build.mjs';
const root = new URL('../', import.meta.url);
const output = new URL('.sentry-artifacts/', root);
await rm(output, { recursive: true, force: true });
await mkdir(output, { recursive: true });
// Keep the exact debug-ID-bearing browser files together, outside either public artifact.
await cp(
	new URL('.svelte-kit/output/client/_app/', root),
	new URL('_app/', output),
	{ recursive: true }
);
// Vite's worker pipeline emits the injected runtime ID but can omit the map's
// metadata stamp. Preserve its mappings and attach the ID from that exact bundle.
async function verifyMaps(directory) {
	for (const item of await readdir(directory, { withFileTypes: true })) {
		const path = new URL(
			item.name + (item.isDirectory() ? '/' : ''),
			directory
		);
		if (item.isDirectory()) {
			await verifyMaps(path);
			continue;
		}
		if (!item.name.endsWith('.js.map')) continue;
		const map = JSON.parse(await readFile(path, 'utf8'));
		const source = await readFile(
			new URL(item.name.slice(0, -4), directory),
			'utf8'
		);
		const id = /\/\/# debugId=([a-f0-9-]{36})(?:\s|$)/.exec(source)?.[1];
		if (!id || (map.debug_id && map.debug_id !== id))
			throw new Error(
				'Missing or mismatched source map debug ID: ' + item.name
			);
		if (!map.debug_id) {
			map.debug_id = id;
			await writeFile(path, JSON.stringify(map));
		}
	}
}
await verifyMaps(output);
await writeFile(
	new URL('release.json', output),
	JSON.stringify({ release: webRelease }) + '\n'
);
async function removeMaps(directory) {
	for (const item of await readdir(directory, { withFileTypes: true })) {
		const path = new URL(
			item.name + (item.isDirectory() ? '/' : ''),
			directory
		);
		if (item.isDirectory()) await removeMaps(path);
		else if (item.name.endsWith('.map')) await rm(path);
	}
}
await removeMaps(new URL('.svelte-kit/output/client/', root));
await removeMaps(new URL('.svelte-kit/cloudflare/', root));
console.log(
	'Private Sentry source maps prepared for ' +
		webRelease +
		'; no upload performed.'
);
