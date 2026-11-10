import { existsSync } from 'node:fs';
import { join } from 'node:path';

// Vite loads these two files in every mode. Require mode-specific inputs so a
// development Firebase/Sentry configuration cannot silently enter a release.
/** @param {string} directory */
export function assertModeSpecificEnv(directory) {
	if (
		['.env', '.env.local'].some((name) => existsSync(join(directory, name)))
	) {
		throw new Error(
			'Use config/.env.development.local or config/.env.production.local; run just config-init to migrate legacy local settings.'
		);
	}
}
