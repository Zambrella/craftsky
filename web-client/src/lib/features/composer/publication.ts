import { withDeadline } from '#lib/async.ts';
import {
	MutationOutcomeUnknownError,
	type CraftskyAPI,
	type Post,
	type PostEditResult,
	type PostPayload,
	type PostEditChanges
} from '#lib/api/client.ts';
import type { PendingWrite } from '../drafts/store';
import { record, str } from './model';
export type PublicationResult = {
	notice: string;
	destination?: string;
	post?: Post;
	edit?: PostEditResult;
};
/** Executes only a previously frozen, durably stored command. Presentation belongs to the editor. */
export async function executePublication(
	api: CraftskyAPI,
	pending: PendingWrite,
	parent: AbortSignal
): Promise<PublicationResult> {
	try {
		return await withDeadline(120000, parent, async (signal) => {
			switch (pending.kind) {
				case 'schedule-create':
					await api.schedules.create(
						pending.key,
						str(pending.body.scheduledAt),
						record(pending.body.payload),
						signal
					);
					return {
						destination: '/scheduled',
						notice: 'Your post has been scheduled.'
					};
				case 'schedule-update':
					await api.schedules.update(
						pending.rkey!,
						str(pending.body.scheduledAt),
						record(pending.body.payload),
						signal
					);
					return {
						destination: '/scheduled',
						notice: 'Your schedule has been updated.'
					};
				case 'schedule-publish': {
					const result = await api.schedules.publish(
						pending.rkey!,
						record(pending.body.payload),
						signal
					);
					return {
						destination: '/scheduled',
						notice:
							result.status === 'publishing'
								? 'Your post is being published.'
								: 'Your post has been published.'
					};
				}
				case 'edit':
					return {
						edit: await api.posts.edit(
							pending.did!,
							pending.rkey!,
							pending.body as unknown as PostEditChanges,
							pending.cid!,
							pending.key,
							signal
						),
						destination: `/posts/${encodeURIComponent(pending.did!)}/${encodeURIComponent(pending.rkey!)}`,
						notice: 'Your changes have been saved.'
					};
				default:
					return {
						post: await api.posts.create(
							pending.body as unknown as PostPayload,
							pending.key,
							signal
						),
						notice: 'Your post has been published.'
					};
			}
		});
	} catch (error) {
		if (
			error instanceof DOMException &&
			['AbortError', 'TimeoutError'].includes(error.name)
		)
			throw new MutationOutcomeUnknownError(pending.key, 1, false, {
				cause: error
			});
		throw error;
	}
}
