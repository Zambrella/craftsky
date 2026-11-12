import { test, expect, type Page } from '@playwright/test';
import { readFile } from 'node:fs/promises';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
async function openCard(page: Page, path = '/profiles/bob.test') {
	await page.goto(path);
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Share card', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Your share card' });
	await expect(dialog.getByRole('img')).toBeVisible();
	const closeBounds = await dialog
		.getByRole('button', { name: 'Close share card' })
		.boundingBox();
	expect(closeBounds!.width).toBe(closeBounds!.height);
	expect(
		await dialog.evaluate(
			(element) => element.scrollWidth <= element.clientWidth
		)
	).toBe(true);
	await expect(dialog).not.toContainText(
		'This browser will save your card as an image.'
	);
	await expect(
		dialog.getByRole('button', { name: 'Share', exact: true })
	).toBeEnabled();
	return dialog;
}
async function artwork(page: Page) {
	return page
		.getByRole('dialog', { name: 'Your share card' })
		.getByRole('img')
		.evaluate(async (element) => {
			const img = element as HTMLImageElement;
			await img.decode();
			const canvas = document.createElement('canvas');
			canvas.width = img.naturalWidth;
			canvas.height = img.naturalHeight;
			canvas.getContext('2d')!.drawImage(img, 0, 0);
			return canvas.toDataURL('image/png');
		});
}

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await context.addCookies([
		{
			name: 'craftsky_account_' + account.id,
			value: Buffer.from(JSON.stringify(account)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
});

test('visitor and own profile export a fixed PNG, restore focus and preserve image across themes and widths', async ({
	page
}, testInfo) => {
	await page.addInitScript(() =>
		Object.defineProperty(navigator, 'canShare', {
			value: () => false,
			configurable: true
		})
	);
	let dialog = await openCard(page);
	const baseline = await artwork(page);
	const button = dialog.getByRole('button', { name: 'Share', exact: true });
	const downloadPromise = page.waitForEvent('download');
	await button.click();
	const download = await downloadPromise;
	expect(download.suggestedFilename()).toBe('CraftSky-profile.png');
	const path = testInfo.outputPath('profile-share-card.png');
	await download.saveAs(path);
	const bytes = await readFile(path);
	expect(bytes.subarray(1, 4).toString()).toBe('PNG');
	expect(bytes.readUInt32BE(16)).toBe(1200);
	expect(bytes.readUInt32BE(20)).toBe(1500);
	const exportedData = `data:image/png;base64,${bytes.toString('base64')}`;
	const effects = await page.evaluate(
		async ({ plain, exported }) => {
			async function pixels(src: string) {
				const img = new Image();
				img.src = src;
				await img.decode();
				const canvas = document.createElement('canvas');
				canvas.width = 1200;
				canvas.height = 1500;
				const ctx = canvas.getContext('2d')!;
				ctx.drawImage(img, 0, 0);
				return ctx.getImageData(0, 0, 1200, 1500).data;
			}
			const [base, decorated] = await Promise.all([
				pixels(plain),
				pixels(exported)
			]);
			let edgeChanges = 0,
				foilChanges = 0,
				qrChanges = 0;
			for (let y = 0; y < 1500; y++)
				for (let x = 0; x < 1200; x++) {
					const index = (y * 1200 + x) * 4;
					if (
						!base
							.slice(index, index + 4)
							.some((value, channel) => value !== decorated[index + channel])
					)
						continue;
					if (x < 15 || x > 1185 || y < 15 || y > 1485) edgeChanges++;
					else foilChanges++;
					if (x >= 350 && x < 850 && y >= 800 && y < 1300) qrChanges++;
				}
			return { edgeChanges, foilChanges, qrChanges };
		},
		{ plain: baseline, exported: exportedData }
	);
	expect(effects.edgeChanges).toBeGreaterThan(0);
	expect(effects.foilChanges).toBeGreaterThan(0);
	expect(effects.qrChanges).toBe(0);
	await dialog.screenshot({
		path: testInfo.outputPath('profile-share-card-desktop.png')
	});
	const source = await dialog.getByRole('img').getAttribute('src');
	await page.keyboard.press('Escape');
	await expect(dialog).not.toBeVisible();
	await expect(
		page.getByLabel('More profile actions', { exact: true })
	).toBeFocused();
	expect(
		await page.evaluate(
			(url) =>
				new Promise<boolean>((resolve) => {
					const img = new Image();
					img.onload = () => resolve(true);
					img.onerror = () => resolve(false);
					img.src = url!;
				}),
			source
		)
	).toBe(false);
	await page.setViewportSize({ width: 320, height: 568 });
	await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'reduce' });
	dialog = await openCard(page);
	await page.addStyleTag({ content: 'html { font-size: 200% !important; }' });
	expect(await artwork(page)).toBe(baseline);
	const secondDownload = page.waitForEvent('download');
	await dialog.getByRole('button', { name: 'Share', exact: true }).click();
	const secondPath = testInfo.outputPath('profile-share-card-mobile.png');
	await (await secondDownload).saveAs(secondPath);
	expect(await readFile(secondPath)).toEqual(bytes);
	const cardBounds = await dialog.getByRole('img').boundingBox();
	const buttonBounds = await dialog
		.getByRole('button', { name: 'Share', exact: true })
		.boundingBox();
	expect(cardBounds!.width / cardBounds!.height).toBeCloseTo(4 / 5);
	expect(buttonBounds!.width).toBeCloseTo(cardBounds!.width);
	await dialog.getByRole('button', { name: 'Close share card' }).click();
	dialog = await openCard(page, '/profiles/alice.test');
	await expect(dialog.getByRole('img')).toHaveAttribute('alt', /alice.test/);
});

test('native image sharing keeps loading visible, supports cancellation, reports safe errors and retries', async ({
	page
}) => {
	await page.addInitScript(() => {
		const state = window as unknown as {
			shareCalls: { name: string; type: string; active: boolean }[];
			finishShare: (outcome: string) => void;
		};
		state.shareCalls = [];
		Object.defineProperty(navigator, 'canShare', {
			value: (data: ShareData) => data.files?.[0].type === 'image/png',
			configurable: true
		});
		Object.defineProperty(navigator, 'share', {
			value: (data: ShareData) => {
				const file = data.files![0];
				state.shareCalls.push({
					name: file.name,
					type: file.type,
					active: navigator.userActivation.isActive
				});
				return new Promise<void>((resolve, reject) => {
					state.finishShare = (outcome) => {
						if (outcome === 'cancel')
							reject(new DOMException('Canceled', 'AbortError'));
						else if (outcome === 'fail')
							reject(new Error('PRIVATE_SHARE_IMAGE_CANARY'));
						else resolve();
					};
				});
			},
			configurable: true
		});
	});
	const dialog = await openCard(page);
	const button = dialog.locator('.share-card-button');
	const idle = await button.boundingBox();
	await button.click();
	await expect(button).toHaveText('Loading...');
	await expect(button).toBeDisabled();
	const busy = await button.boundingBox();
	expect(busy!.width).toBe(idle!.width);
	expect(busy!.height).toBe(idle!.height);
	await expect(button).toHaveCSS('background-color', 'rgb(245, 239, 228)');
	expect(
		await page.evaluate(
			() => (window as unknown as { shareCalls: unknown[] }).shareCalls
		)
	).toEqual([
		{ name: 'CraftSky-profile.png', type: 'image/png', active: true }
	]);
	const finish = (value: string) =>
		page.evaluate(
			(outcome) =>
				(
					window as unknown as { finishShare: (outcome: string) => void }
				).finishShare(outcome),
			value
		);
	await finish('cancel');
	await expect(button).toHaveText('Share');
	await expect(dialog.getByRole('alert')).not.toBeVisible();
	await button.click();
	await finish('fail');
	await expect(dialog.getByRole('alert')).toHaveText(
		'Could not complete this request. Please try again.'
	);
	await expect(dialog).not.toContainText('PRIVATE_SHARE_IMAGE_CANARY');
	await button.click();
	await finish('success');
	await expect(button).toBeEnabled();
	await expect(dialog.getByRole('alert')).not.toBeVisible();
});

test('a render failure can be retried without an unhandled error', async ({
	page
}) => {
	await page.route('**/profile-share-mark.png', (route) => route.abort());
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.name));
	await page.goto('/profiles/bob.test');
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Share card', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Your share card' });
	await expect(dialog.getByRole('button', { name: 'Try again' })).toBeEnabled();
	await page.unroute('**/profile-share-mark.png');
	await dialog.getByRole('button', { name: 'Try again' }).click();
	await expect(dialog.getByRole('img')).toBeVisible();
	expect(errors).toEqual([]);
});

test('avatar without CDN CORS headers renders through the restricted same-origin endpoint', async ({
	page,
	request
}) => {
	const avatar =
		'https://cdn.bsky.app/img/avatar/plain/did:plc:abcdefghijklmnopqrstuvwx/bafkreitest@jpeg';
	const response = await request.post('http://127.0.0.1:19847/__test/control', {
		data: { shareAvatar: avatar }
	});
	expect(response.ok()).toBe(true);
	// The test's same-origin endpoint response stands in for the CDN bytes;
	// the independently tested server boundary never forwards credentials.
	await page.route('**/api/profile-share-avatar?*', async (route) => {
		expect(new URL(route.request().url()).searchParams.get('url')).toBe(avatar);
		const png = await page.request.get('/app-icon.png');
		await route.fulfill({ contentType: 'image/png', body: await png.body() });
	});
	const dialog = await openCard(page);
	const pixel = await dialog.getByRole('img').evaluate(async (element) => {
		const img = element as HTMLImageElement;
		await img.decode();
		const canvas = document.createElement('canvas');
		canvas.width = 1200;
		canvas.height = 1500;
		const ctx = canvas.getContext('2d')!;
		ctx.drawImage(img, 0, 0);
		return [...ctx.getImageData(600, 230, 1, 1).data];
	});
	// The avatar interior contains the supplied image, rather than lilac initials.
	expect(pixel.slice(0, 3)).not.toEqual([201, 185, 229]);
});

test('web foil shader and pointer tilt preserve the PNG and settle after release', async ({
	page
}, testInfo) => {
	const failures: string[] = [];
	page.on('pageerror', (error) => failures.push(error.name));
	const dialog = await openCard(page);
	const stage = dialog.locator('.card-stage');
	const card = dialog.locator('.artwork');
	await expect(stage).toHaveAttribute('data-foil', 'webgl');
	const original = await artwork(page);
	const bounds = await stage.boundingBox();
	const x = bounds!.x + bounds!.width / 2,
		y = bounds!.y + bounds!.height / 2;
	await page.mouse.move(x, y);
	await page.mouse.down();
	await page.mouse.move(x + 85, y - 85, { steps: 8 });
	await expect
		.poll(() => card.evaluate((node) => node.style.transform))
		.not.toBe('perspective(900px) rotateX(0deg) rotateY(0deg)');
	await expect(stage).toHaveClass(/dragging/);
	expect(await artwork(page)).toBe(original);
	// Read within the render frame, before WebGL's default drawing buffer clears.
	const material = await stage.evaluate(async (element) => {
		const rect = element.getBoundingClientRect();
		element.dispatchEvent(
			new PointerEvent('pointermove', {
				pointerType: 'mouse',
				clientX: rect.right - 10,
				clientY: rect.top + 10
			})
		);
		return new Promise<{ visible: boolean; qrClear: boolean }>((resolve) =>
			requestAnimationFrame(() =>
				requestAnimationFrame(() => {
					const canvas = element.querySelector('canvas')!;
					const gl = canvas.getContext('webgl')!;
					const bytes = new Uint8Array(canvas.width * canvas.height * 4);
					gl.readPixels(
						0,
						0,
						canvas.width,
						canvas.height,
						gl.RGBA,
						gl.UNSIGNED_BYTE,
						bytes
					);
					let visible = false,
						qrClear = true;
					for (let py = 0; py < canvas.height; py++)
						for (let px = 0; px < canvas.width; px++) {
							const alpha = bytes[(py * canvas.width + px) * 4 + 3];
							if (alpha) visible = true;
							const cx = (px / canvas.width) * 480,
								cy = (1 - py / canvas.height) * 600;
							if (cx > 145 && cx < 335 && cy > 325 && cy < 515 && alpha)
								qrClear = false;
						}
					resolve({ visible, qrClear });
				})
			)
		);
	});
	expect(material).toEqual({ visible: true, qrClear: true });
	await dialog.screenshot({
		path: testInfo.outputPath('profile-card-tilted.png')
	});
	await page.mouse.up();
	await page.mouse.move(bounds!.x - 10, y);
	await expect
		.poll(() => card.evaluate((node) => node.style.transform))
		.toBe('perspective(900px) rotateX(0deg) rotateY(0deg)');
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.mouse.move(x + 50, y - 50);
	expect(await card.evaluate((node) => node.style.transform)).toBe(
		'perspective(900px) rotateX(0deg) rotateY(0deg)'
	);
	expect(await artwork(page)).toBe(original);
	await dialog.getByRole('button', { name: 'Close share card' }).click();
	expect(failures).toEqual([]);
});

test('no WebGL falls back to the complete card and sharing still works', async ({
	page
}) => {
	await page.addInitScript(() => {
		const original = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function (
			this: HTMLCanvasElement,
			type: string,
			...args: unknown[]
		) {
			if (type === 'webgl') return null;
			return Reflect.apply(original, this, [type, ...args]);
		} as typeof original;
	});
	const dialog = await openCard(page);
	await expect(dialog.locator('.card-stage')).toHaveAttribute(
		'data-foil',
		'static'
	);
	await expect(dialog.getByRole('img')).toBeVisible();
	await expect(
		dialog.getByRole('button', { name: 'Share', exact: true })
	).toBeEnabled();
});

test('touch dragging captures movement and cancellation returns the card to rest', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const input = await page.context().newCDPSession(page);
	await input.send('Emulation.setTouchEmulationEnabled', {
		enabled: true,
		maxTouchPoints: 1
	});
	const dialog = await openCard(page);
	const stage = dialog.locator('.card-stage');
	const bounds = await stage.boundingBox();
	const x = bounds!.x + bounds!.width / 2,
		y = bounds!.y + bounds!.height / 2;
	await input.send('Input.dispatchTouchEvent', {
		type: 'touchStart',
		touchPoints: [{ x, y }]
	});
	await input.send('Input.dispatchTouchEvent', {
		type: 'touchMove',
		touchPoints: [{ x: x + 55, y: y - 55 }]
	});
	await expect(stage).toHaveClass(/dragging/);
	await expect
		.poll(() =>
			dialog.locator('.artwork').evaluate((node) => node.style.transform)
		)
		.not.toBe('perspective(900px) rotateX(0deg) rotateY(0deg)');
	await input.send('Input.dispatchTouchEvent', {
		type: 'touchCancel',
		touchPoints: []
	});
	await expect(stage).not.toHaveClass(/dragging/);
	await expect
		.poll(() =>
			dialog.locator('.artwork').evaluate((node) => node.style.transform)
		)
		.toBe('perspective(900px) rotateX(0deg) rotateY(0deg)');
	expect(
		await dialog.evaluate((node) => node.scrollWidth <= node.clientWidth)
	).toBe(true);
	await input.detach();
});
