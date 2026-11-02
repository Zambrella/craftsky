import { defineEnvVars } from '@sveltejs/kit/env';
const optionalPublic = {
	public: true,
	static: true,
	schema: (value: string | undefined) => value?.trim() || ''
};
export const variables = defineEnvVars({
	PUBLIC_FIREBASE_API_KEY: optionalPublic,
	PUBLIC_FIREBASE_APP_ID: optionalPublic,
	PUBLIC_FIREBASE_PROJECT_ID: optionalPublic,
	PUBLIC_FIREBASE_MESSAGING_SENDER_ID: optionalPublic,
	PUBLIC_FIREBASE_VAPID_KEY: optionalPublic,
	PUBLIC_FIREBASE_AUTH_DOMAIN: optionalPublic,
	PUBLIC_SENTRY_DSN: optionalPublic
});
