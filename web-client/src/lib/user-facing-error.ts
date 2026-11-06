import { APIError } from './api/transport';
import { apiErrorMessage } from './api/error-message';

/** Only reviewed, client-authored UI copy belongs here. Never wrap dependency prose. */
export class UserFacingError extends Error {
	readonly name = 'UserFacingError';
}
/** A user can correct this input; it is not an unexpected application failure. */
export class InputError extends UserFacingError {}
export function userErrorMessage(error: unknown): string {
	if (error instanceof APIError) return apiErrorMessage(error);
	if (error instanceof UserFacingError) return error.message;
	return 'Something went wrong. Please try again.';
}
