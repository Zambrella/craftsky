type PrivateStorage = Pick<Storage, 'length' | 'key' | 'removeItem'>;
/** Removes only this browser activation's private chat state; shared preferences and other accounts remain. */
export function clearAccountChat(
	storage: PrivateStorage,
	accountId: string,
	did: string
): void {
	const prefix = `craftsky.chat.v1:${encodeURIComponent(accountId + ':' + did)}:`;
	const keys = Array.from({ length: storage.length }, (_, index) =>
		storage.key(index)
	).filter((key): key is string => !!key && key.startsWith(prefix));
	for (const key of keys) storage.removeItem(key);
}
