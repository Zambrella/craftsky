import type { RequestHandler } from './$types';
import { profileShareAvatar } from '#lib/server/profile-share-avatar.ts';

export const GET: RequestHandler = ({ url }) =>
	profileShareAvatar(url.searchParams.get('url'));
