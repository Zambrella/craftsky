/** Hard bounds are independent of the archive's untrusted size declarations. */
export const bzipLimits = Object.freeze({
	outputBytes: 20 * 1024 * 1024,
	compressedBytes: 40 * 1024 * 1024,
	deadlineMs: 30000
});

export function assertBzipSize(size: number): void {
	if (!Number.isSafeInteger(size) || size < 0 || size > bzipLimits.outputBytes)
		throw new Error('Invalid BZip2 output size.');
}
