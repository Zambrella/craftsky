import { sharedCraftskyLink } from './model';
// The public link handoff survives a client-side route component replacement.
// Unsent message text is never put here or persisted during recipient selection.
const links = new Map<string, { link: string; expires: number }>();
const key = (account: string, conversation: string) =>
	JSON.stringify([account, conversation]);
export function stageSharedLink(
	account: string,
	conversation: string,
	raw: string
): void {
	const link = sharedCraftskyLink(raw);
	if (!link) return;
	for (const [id, item] of links)
		if (item.expires <= Date.now()) links.delete(id);
	if (links.size >= 8) links.delete(links.keys().next().value!);
	links.set(key(account, conversation), { link, expires: Date.now() + 60000 });
}
export function takeSharedLink(
	account: string,
	conversation: string
): string | undefined {
	const id = key(account, conversation),
		item = links.get(id);
	links.delete(id);
	return item && item.expires > Date.now() ? item.link : undefined;
}
