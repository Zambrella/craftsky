import type { HandleClientError, ClientInit } from '@sveltejs/kit/hooks';
import {
	applyConsent,
	reportFailure
} from '#lib/features/diagnostics/client.ts';
import { installLocalDiagnostics } from '#lib/features/diagnostics/local.ts';

// Install protection before hydration, including failures outside the root layout.
export const init: ClientInit = async () => {
	const remove = installLocalDiagnostics();
	import.meta.hot?.dispose(remove);
	await applyConsent();
};

export const handleError: HandleClientError = ({ error, kind }) => {
	if (kind === 'unknown') void reportFailure(error, 'framework');
};
