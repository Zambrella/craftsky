import { afterEach, expect, it, vi } from 'vitest';
import { prepareImage } from './media.ts';
import { InputError, UserFacingError } from '#lib/user-facing-error.ts';
import { expectedFailure } from '../diagnostics/privacy.ts';

afterEach(() => vi.unstubAllGlobals());

it('explains an unreadable image as correctable input and keeps the decoder cause', async () => {
	const cause = new DOMException('PRIVATE_IMAGE_CANARY', 'InvalidStateError');
	vi.stubGlobal('createImageBitmap', vi.fn().mockRejectedValue(cause));
	const failure = await prepareImage(
		new Blob(['invalid']),
		'private-name'
	).catch((error) => error);
	expect(failure).toBeInstanceOf(InputError);
	expect(failure.message).toContain('Try another image');
	expect(failure.cause).toBe(cause);
	expect(expectedFailure(failure)).toBe(true);
	expect(failure.message).not.toContain('PRIVATE_IMAGE_CANARY');
});

it('keeps unexpected image decoder failures reportable with safe recovery copy', async () => {
	const cause = new RangeError('PRIVATE_IMAGE_CANARY');
	vi.stubGlobal('createImageBitmap', vi.fn().mockRejectedValue(cause));
	const failure = await prepareImage(new Blob(['image']), 'private-name').catch(
		(error) => error
	);
	expect(failure).toBeInstanceOf(UserFacingError);
	expect(failure.cause).toBe(cause);
	expect(failure.message).toContain('Try another image');
	expect(expectedFailure(failure)).toBe(false);
	expect(failure.message).not.toContain('PRIVATE_IMAGE_CANARY');
});
