/** Only AppView-generated UUIDv4 values may be used for request correlation. */
export function appViewRequestId(value: unknown): string | undefined {
	return typeof value === 'string' &&
		/^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/i.test(
			value
		)
		? value
		: undefined;
}
