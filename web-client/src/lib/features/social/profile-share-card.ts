import QRCode from 'qrcode';
import type { Profile } from '#lib/api/client.ts';
import { label, origin, safeURL } from '#lib/public.ts';

export const shareCardSize = { width: 1200, height: 1500 } as const;
const blue = '#1535d6',
	paper = '#f5efe4',
	ink = '#161210';
const crafts = ['sewing', 'knitting', 'crochet', 'quilting', 'embroidery'];
const swatches = ['#f6d966', '#c9b9e5', '#9dc4dc', '#c8d3ae', '#e8a590'];
export function shareCardAvatarURL(value?: string | null) {
	const safe = safeURL(value ?? undefined);
	if (!safe) return undefined;
	const url = new URL(safe);
	return url.origin === 'https://cdn.bsky.app'
		? `/api/profile-share-avatar?url=${encodeURIComponent(safe)}`
		: safe;
}

export const profileShareURL = (did: string) =>
	`${origin}/profiles/${encodeURIComponent(did)}`;

function image(src: string): Promise<HTMLImageElement> {
	return new Promise((resolve, reject) => {
		const img = new Image();
		img.crossOrigin = 'anonymous';
		const timer = setTimeout(() => {
			img.src = '';
			reject(new Error('Share card image timed out'));
		}, 8000);
		img.onload = () => {
			clearTimeout(timer);
			resolve(img);
		};
		img.onerror = () => {
			clearTimeout(timer);
			reject(new Error('Share card image unavailable'));
		};
		img.src = src;
	});
}

function shape(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	w: number,
	h: number,
	r: number,
	fill: string
) {
	ctx.beginPath();
	ctx.roundRect(x, y, w, h, r);
	ctx.fillStyle = fill;
	ctx.fill();
}

function text(
	ctx: CanvasRenderingContext2D,
	value: string,
	y: number,
	size: number,
	serif = false,
	maxWidth = 408
) {
	ctx.font = `${size}px ${serif ? '"DM Serif Display"' : 'Outfit'}`;
	const width = ctx.measureText(value).width;
	if (width > maxWidth)
		ctx.font = `${(size * maxWidth) / width}px ${serif ? '"DM Serif Display"' : 'Outfit'}`;
	ctx.fillStyle = paper;
	ctx.textAlign = 'center';
	ctx.textBaseline = 'middle';
	ctx.fillText(value, 240, y);
}

/** A single canvas is both the preview and export: no theme, zoom or DPR changes the PNG. */
export async function renderProfileShareCard(
	profile: Pick<Profile, 'did' | 'handle' | 'displayName' | 'avatar' | 'crafts'>
): Promise<File> {
	const name = profile.displayName?.trim() || profile.handle;
	const [logo, mark, icons, avatar] = await Promise.all([
		image('/app-icon.png'),
		image('/profile-share-mark.png'),
		Promise.all(crafts.map((craft) => image(`/craft_icons/${craft}.svg`))),
		shareCardAvatarURL(profile.avatar)
			? image(shareCardAvatarURL(profile.avatar)!).catch(() => null)
			: null,
		document.fonts.load('32px "DM Serif Display"', name),
		document.fonts.load('14px Outfit', `@${profile.handle}`),
		document.fonts.load('600 14px Outfit', profile.crafts.map(label).join(' '))
	]);
	const canvas = document.createElement('canvas');
	canvas.width = shareCardSize.width;
	canvas.height = shareCardSize.height;
	const ctx = canvas.getContext('2d');
	if (!ctx) throw new Error('Share card canvas unavailable');
	ctx.scale(2.5, 2.5);
	ctx.save();
	ctx.beginPath();
	ctx.roundRect(0, 0, 480, 600, 22);
	ctx.clip();
	ctx.fillStyle = blue;
	ctx.fillRect(0, 0, 480, 600);
	// Static, faint craft sequence. The web card deliberately has no motion or shader.
	ctx.globalAlpha = 0.08;
	ctx.filter = 'invert(1)';
	for (let row = 0; row < 10; row++)
		for (let col = 0; col < 8; col++) {
			ctx.drawImage(
				icons[(row * 3 + col) % icons.length],
				col * 70 - (row % 2) * 35,
				row * 70 - 10,
				30,
				30
			);
		}
	ctx.globalAlpha = 1;
	ctx.filter = 'none';
	ctx.strokeStyle = '#ffffff77';
	ctx.lineWidth = 1.5;
	ctx.beginPath();
	ctx.roundRect(1, 1, 478, 598, 21);
	ctx.stroke();
	ctx.drawImage(logo, 176, 24, 26, 26);
	ctx.fillStyle = paper;
	ctx.textAlign = 'left';
	ctx.textBaseline = 'middle';
	ctx.font = '24px "DM Serif Display"';
	ctx.fillText('CraftSky', 210, 37);
	ctx.beginPath();
	ctx.arc(240, 120, 48, 0, Math.PI * 2);
	ctx.fillStyle = paper;
	ctx.fill();
	ctx.save();
	ctx.beginPath();
	ctx.arc(240, 120, 42, 0, Math.PI * 2);
	ctx.clip();
	ctx.fillStyle = '#c9b9e5';
	ctx.fillRect(198, 78, 84, 84);
	if (avatar) {
		const side = Math.min(avatar.naturalWidth, avatar.naturalHeight);
		ctx.drawImage(
			avatar,
			(avatar.naturalWidth - side) / 2,
			(avatar.naturalHeight - side) / 2,
			side,
			side,
			198,
			78,
			84,
			84
		);
	} else {
		ctx.fillStyle = ink;
		ctx.font = '40px "DM Serif Display"';
		ctx.textAlign = 'center';
		ctx.fillText(Array.from(name)[0]?.toUpperCase() || '?', 240, 122);
	}
	ctx.restore();
	text(ctx, name, 201, 32, true);
	text(ctx, `@${profile.handle}`, 229, 14);
	const selected = [
		...new Set(
			profile.crafts.map((craft) => craft.split('#').at(-1)!.toLowerCase())
		)
	];
	// Measure the whole craft block, then scale it uniformly to fit the fixed slot.
	ctx.font = '600 14px Outfit';
	const pills = selected.map((craft, index) => ({
		craft,
		index,
		width:
			ctx.measureText(label(craft)).width + (crafts.includes(craft) ? 44 : 22)
	}));
	const rows: (typeof pills)[] = [[]];
	let rowWidth = 0;
	for (const pill of pills) {
		if (rowWidth && rowWidth + pill.width + 8 > 408) {
			rows.push([]);
			rowWidth = 0;
		}
		rows.at(-1)!.push(pill);
		rowWidth += pill.width + 8;
	}
	const blockWidth = Math.max(
		1,
		...rows.map((row) => row.reduce((n, pill) => n + pill.width + 8, -8))
	);
	const scale = Math.min(1, 408 / blockWidth, 48 / (rows.length * 34 - 8));
	ctx.save();
	ctx.translate(240, 280);
	ctx.scale(scale, scale);
	rows.forEach((row, r) => {
		let x = -row.reduce((n, pill) => n + pill.width + 8, -8) / 2;
		const y = r * 34 - (rows.length * 34 - 8) / 2;
		row.forEach(({ craft, index, width }) => {
			shape(ctx, x, y, width, 26, 13, swatches[index % swatches.length]);
			ctx.strokeStyle = ink;
			ctx.lineWidth = 1.5;
			ctx.stroke();
			const icon = icons[crafts.indexOf(craft)];
			if (icon) ctx.drawImage(icon, x + 9, y + 5, 16, 16);
			ctx.fillStyle = ink;
			ctx.font = '600 14px Outfit';
			ctx.textAlign = 'left';
			ctx.fillText(label(craft), x + (icon ? 30 : 11), y + 13);
			x += width + 8;
		});
	});
	ctx.restore();
	shape(ctx, 140, 320, 200, 200, 14, '#fff');
	const qr = QRCode.create(profileShareURL(profile.did), {
		errorCorrectionLevel: 'H'
	});
	const modules = qr.modules.size,
		cell = 176 / modules;
	ctx.fillStyle = blue;
	for (let row = 0; row < modules; row++)
		for (let col = 0; col < modules; col++) {
			if (qr.modules.get(row, col)) {
				const x = Math.round((152 + col * cell) * 2.5) / 2.5;
				const y = Math.round((332 + row * cell) * 2.5) / 2.5;
				const right = Math.round((152 + (col + 1) * cell) * 2.5) / 2.5;
				const bottom = Math.round((332 + (row + 1) * cell) * 2.5) / 2.5;
				ctx.fillRect(x, y, right - x, bottom - y);
			}
		}
	ctx.fillStyle = '#fff';
	ctx.fillRect(223, 403, 34, 34);
	// Crop the adaptive-icon safe area, retaining its transparent mark on white.
	const crop = (mark.naturalWidth * 34) / 48;
	ctx.drawImage(
		mark,
		(mark.naturalWidth - crop) / 2,
		(mark.naturalHeight - crop) / 2,
		crop,
		crop,
		223,
		403,
		34,
		34
	);
	text(ctx, 'Meet me on CraftSky', 556, 14);
	ctx.restore();
	const blob = await new Promise<Blob>((resolve, reject) =>
		canvas.toBlob(
			(value) =>
				value
					? resolve(value)
					: reject(new Error('Share card PNG encoding failed')),
			'image/png'
		)
	);
	return new File([blob], 'CraftSky-profile.png', { type: 'image/png' });
}

export function canShareCard(file: File) {
	return !!navigator.share && !!navigator.canShare?.({ files: [file] });
}

/** Called directly from the click handler so native sharing retains user activation. */
export async function shareProfileCard(file: File, url: string) {
	if (canShareCard(file)) {
		await navigator.share({ files: [file] });
		return;
	}
	const link = document.createElement('a');
	link.href = url;
	link.download = file.name;
	link.click();
}
