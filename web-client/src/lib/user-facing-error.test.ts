import { describe, it, expect } from 'vitest';
import {
	InputError,
	UserFacingError,
	userErrorMessage
} from './user-facing-error';
import { FieldValidationError } from './field-validation';
import { expectedFailure } from './features/diagnostics/privacy';
import { message, parseFollowing } from './features/settings/model';
import { APIError } from './api/transport';

describe('client-owned error presentation', () => {
	it('does not display dependency prose, even for errors with a forged name', () => {
		const error = new Error('PRIVATE_CANARY');
		error.name = 'UserFacingError';
		expect(userErrorMessage(error)).toBe(
			'Something went wrong. Please try again.'
		);
		expect(message(error)).not.toContain('PRIVATE_CANARY');
		expect(message(new APIError(503, 'failed', 'PRIVATE_CANARY'))).toBe(
			'CraftSky is having trouble right now. Please try again.'
		);
	});
	it('keeps reviewed validation and recovery messages while classifying them separately', () => {
		const validation = new FieldValidationError('title', 'Enter a title.');
		expect(userErrorMessage(validation)).toBe('Enter a title.');
		expect(expectedFailure(validation)).toBe(true);
		expect(expectedFailure(new InputError('Choose an image.'))).toBe(true);
		const failure = new UserFacingError('Local storage is unavailable.', {
			cause: new Error('PRIVATE')
		});
		expect(userErrorMessage(failure)).toBe('Local storage is unavailable.');
		expect(expectedFailure(failure)).toBe(false);
	});
	it('retains actionable import validation instead of dependency text', () => {
		try {
			parseFollowing('{PRIVATE_CANARY', true);
		} catch (error) {
			expect(message(error)).toContain('not valid JSON');
			expect(message(error)).not.toContain('PRIVATE_CANARY');
		}
	});
});
