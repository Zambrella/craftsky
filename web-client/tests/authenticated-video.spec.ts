import { test, expect } from '@playwright/test';
import { resolve } from 'node:path';
const fixture = 'http://127.0.0.1:19847';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ page, context }) => {
	await fetch(fixture + '/__test/reset');
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
		},
		{
			name: 'craftsky_active',
			value: account.id,
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
});
for (const failed of [false, true]) {
	test(`MOV uploads through Bunny and ${failed ? 'keeps the draft on processing failure' : 'publishes the processed MP4 proof'}`, async ({
		page
	}) => {
		if (failed)
			await fetch(fixture + '/__test/control', {
				method: 'POST',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify({ owners: { alice: { videoFailure: true } } })
			});
		const uploaded: Buffer[] = [];
		await page.route('https://video.bsky.app/**', () => {
			throw new Error('Bunny capability must never reach Bluesky');
		});
		await page.route('https://video.bunnycdn.com/**', async (route) => {
			const request = route.request();
			expect(request.headers().cookie).toBeUndefined();
			expect(request.headers().authorization).toBeUndefined();
			expect(request.headers().authorizationsignature).toBe('c'.repeat(64));
			const headers = {
				'access-control-allow-origin': '*',
				'access-control-expose-headers': 'Location, Upload-Offset'
			};
			if (request.method() === 'POST') {
				expect(request.headers()['upload-metadata']).toContain('collection ');
				await route.fulfill({
					status: 201,
					headers: {
						...headers,
						location: 'https://video.bunnycdn.com/tusupload/fixture'
					}
				});
			} else {
				expect(request.method()).toBe('PATCH');
				uploaded.push(request.postDataBuffer()!);
				await route.fulfill({
					status: 204,
					headers: {
						...headers,
						'upload-offset': String(
							uploaded.reduce((sum, part) => sum + part.length, 0)
						)
					}
				});
			}
		});
		await page.goto('/compose');
		await page
			.getByRole('textbox', {
				name: 'What would you like to share?',
				exact: true
			})
			.fill('Bunny MOV browser publication');
		const source = Buffer.from([
			0,
			0,
			0,
			20,
			...Buffer.from('ftypqt  '),
			0,
			0,
			0,
			0
		]);
		await page.getByLabel('Choose a video', { exact: true }).setInputFiles({
			name: 'private-source.mov',
			mimeType: 'video/quicktime',
			buffer: source
		});
		await expect(
			page.getByRole('heading', { name: 'Video details', exact: true })
		).toBeVisible();
		await expect(
			page.getByText(
				'Preview unavailable in this browser. You can still upload this video.'
			)
		).toBeVisible();
		await page
			.getByLabel('Video description', { exact: true })
			.fill('A knitting demonstration');
		await page
			.getByRole('button', { name: 'Back to post', exact: true })
			.click();
		await expect(
			page.getByRole('button', { name: 'Inspect selected video', exact: true })
		).toBeVisible();
		await page.getByRole('button', { name: 'Post', exact: true }).click();
		if (failed) {
			await expect(page.getByText(/Video processing failed/)).toBeVisible();
			await expect(
				page.getByRole('textbox', {
					name: 'What would you like to share?',
					exact: true
				})
			).toHaveValue('Bunny MOV browser publication');
		} else await expect(page).toHaveURL(/\/feed$/);
		expect(Buffer.concat(uploaded)).toEqual(source);
		const calls = await (await fetch(fixture + '/__test/requests')).json();
		const status = calls.find(
			(call: any) =>
				call.path.endsWith('/status') && call.path.includes('/blobs/videos/')
		);
		expect(status).toMatchObject({ method: 'POST', owner: account.did });
		const posts = calls.filter(
			(call: any) => call.path === '/v1/posts' && call.method === 'POST'
		);
		if (failed) expect(posts).toHaveLength(0);
		else {
			expect(posts).toHaveLength(1);
			expect(posts[0].body.embed).toMatchObject({
				video: {
					blob: {
						mimeType: 'video/mp4',
						ref: { $link: 'bafy-processed-video' }
					},
					alt: 'A knitting demonstration',
					jobId: status.path.split('/')[4]
				}
			});
			expect(JSON.stringify(posts[0].body)).not.toContain('private-source.mov');
		}
	});
}

// Synthetic blue H.264 fixture, generated with FFmpeg; no user media.
test('video picture toggles playback and buffering uses the primary colour', async ({
	page
}) => {
	const host = 'https://video-fixture.craftsky.test/';
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.items[0].post.images = [];
		body.items[0].post.video = {
			cid: 'bafy-video',
			mime: 'video/mp4',
			size: 100,
			alt: 'Test video',
			playlist: host + 'playlist.m3u8'
		};
		await route.fulfill({ response, json: body });
	});
	let release!: () => void;
	const held = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route(host + '**', async (route) => {
		const name = new URL(route.request().url()).pathname.slice(1);
		if (name.endsWith('.mpegts')) await held;
		await route.fulfill({
			path: resolve('tests/fixtures/video', name),
			contentType: name.endsWith('.m3u8')
				? 'application/vnd.apple.mpegurl'
				: 'video/mp2t',
			headers: { 'access-control-allow-origin': '*' }
		});
	});
	await page.goto('/feed');
	const player = page.locator('.video-player').first();
	const video = player.locator('video');
	await player
		.getByRole('button', { name: 'Play video', exact: true })
		.click({ position: { x: 25, y: 25 } });
	const loader = player.getByRole('status', { name: 'Loading video' });
	await expect(loader).toBeVisible();
	const spinner = loader.locator('.spinner');
	expect(
		await spinner.evaluate((node) => getComputedStyle(node).borderTopColor)
	).toBe(
		await spinner.evaluate((node) => {
			const probe = document.createElement('span');
			probe.style.color = 'var(--primary)';
			node.append(probe);
			const value = getComputedStyle(probe).color;
			probe.remove();
			return value;
		})
	);
	release();
	await expect
		.poll(() => video.evaluate((node) => (node as HTMLVideoElement).paused))
		.toBe(false);
	await expect(loader).toHaveCount(0);
	await player
		.getByRole('button', { name: 'Pause video', exact: true })
		.click({ position: { x: 35, y: 25 } });
	await expect
		.poll(() => video.evaluate((node) => (node as HTMLVideoElement).paused))
		.toBe(true);
	await player
		.getByRole('button', { name: 'Play video', exact: true })
		.press('Enter');
	await expect
		.poll(() => video.evaluate((node) => (node as HTMLVideoElement).paused))
		.toBe(false);
	await expect(video).toHaveAttribute('controls', '');
	expect(
		await video.evaluate((node) => ({
			muted: (node as HTMLVideoElement).muted,
			volume: (node as HTMLVideoElement).volume
		}))
	).toEqual({ muted: false, volume: 1 });
	await expect(page).toHaveURL(/\/feed$/);
});

const bunnyEmbed =
	'https://player.mediadelivery.net/embed/123/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa';
async function hostedVideo(
	page: import('@playwright/test').Page,
	extra: Record<string, unknown> = {}
) {
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch(),
			body = await response.json();
		body.items[0].post.images = [];
		body.items[0].post.video = {
			cid: 'bafy-video',
			mime: 'video/mp4',
			size: 100,
			alt: 'Hosted test video',
			playlist: 'https://video-fixture.craftsky.test/playlist.m3u8',
			embedUrl: bunnyEmbed,
			aspectRatio: { width: 9, height: 16 },
			...extra
		};
		await route.fulfill({ response, json: body });
	});
}
const playerFixture = `<!doctype html><html><body style="margin:0;background:black;color:white"><button id="play">Play video</button><button id="full">Fullscreen</button><script>
 const origin=new URL(document.referrer).origin;
 const emit=(event,value)=>parent.postMessage(JSON.stringify({context:'player.js',version:'0.0.11',event,value}),origin);
 const ready=()=>emit('ready',{src:location.href});
 let playing=false;
 const button=document.querySelector('#play');
 const pause=()=>{playing=false;button.textContent='Play video';};
 button.onclick=()=>{playing=!playing;button.textContent=playing?'Pause video':'Play video';};
 document.querySelector('#full').onclick=()=>document.documentElement.requestFullscreen();
 addEventListener('message',event=>{if(event.source!==parent||event.origin!==origin)return;const data=JSON.parse(event.data);if(data.method==='addEventListener'&&data.value==='ready')ready();if(data.method==='pause')pause();});
 ready();
 </script></body></html>`;

test('Bunny iframe is lazy, supports fullscreen and pauses when scrolled out of view', async ({
	page
}) => {
	await hostedVideo(page);
	await page.route('https://player.mediadelivery.net/**', (route) =>
		route.fulfill({ contentType: 'text/html', body: playerFixture })
	);
	await page.goto('/feed');
	const frame = page.locator('.bunny-video iframe').first();
	await expect(frame).toBeVisible();
	await expect(frame).toHaveAttribute('loading', 'lazy');
	await expect(frame).toHaveAttribute('allowfullscreen', '');
	const url = new URL((await frame.getAttribute('src'))!);
	expect(url.searchParams.get('autoplay')).toBe('false');
	expect(url.searchParams.get('muted')).toBe('false');
	expect(url.searchParams.get('preload')).toBe('false');
	expect(url.searchParams.get('rememberSettings')).toBe('false');
	await expect(page.locator('.bunny-video [role=status]')).toHaveCount(0);
	const inside = frame.contentFrame();
	await inside.getByRole('button', { name: 'Play video', exact: true }).click();
	await expect(
		inside.getByRole('button', { name: 'Pause video', exact: true })
	).toBeVisible();
	await inside.getByRole('button', { name: 'Fullscreen', exact: true }).click();
	await expect
		.poll(() => page.evaluate(() => !!document.fullscreenElement))
		.toBe(true);
	await page.evaluate(() => document.exitFullscreen());
	await expect(
		inside.getByRole('button', { name: 'Pause video', exact: true })
	).toBeVisible();
	await frame.evaluate((node) => {
		node.style.position = 'fixed';
		node.style.top = '-3000px';
	});
	await expect(
		inside.getByRole('button', { name: 'Play video', exact: true })
	).toBeAttached();
	await page.goto('/saved');
	await expect(page.locator('.bunny-video iframe')).toHaveCount(0);
});

test('Bunny player has spacing above it and no standard-player button', async ({
	page
}) => {
	await hostedVideo(page);
	await page.route('https://player.mediadelivery.net/**', (route) =>
		route.fulfill({ contentType: 'text/html', body: playerFixture })
	);
	await page.goto('/feed');
	await expect(page.locator('.bunny-video iframe')).toHaveCount(1);
	await expect(
		page.getByRole('button', { name: 'Use standard player', exact: true })
	).toHaveCount(0);
	await expect(page.locator('.post-video').first()).toHaveCSS(
		'margin-top',
		'8px'
	);
});

test('PDS caption tracks retain the standard player', async ({ page }) => {
	await hostedVideo(page, {
		captions: [{ cid: 'bafy-caption', lang: 'en', uri: '/v1/caption' }]
	});
	await page.goto('/feed');
	await expect(page.locator('.bunny-video iframe')).toHaveCount(0);
	await expect(page.locator('.video-player video')).toHaveCount(1);
});

test('real development Bunny player initializes and plays the synthetic fixture', async ({
	page
}) => {
	test.skip(
		!process.env.BUNNY_PLAYER_TEST_EMBED_URL,
		'Opt-in synthetic development fixture only.'
	);
	await hostedVideo(page, {
		embedUrl: process.env.BUNNY_PLAYER_TEST_EMBED_URL
	});
	await page.goto('/feed');
	const frame = page.locator('.bunny-video iframe');
	await expect(frame).toBeVisible();
	await expect(page.locator('.bunny-video [role=status]')).toHaveCount(0, {
		timeout: 30000
	});
	const inside = frame.contentFrame();
	await inside.getByRole('button', { name: /^play/i }).first().click();
	await expect
		.poll(
			() =>
				inside
					.locator('video')
					.evaluate((node) => (node as HTMLVideoElement).paused),
			{ timeout: 30000 }
		)
		.toBe(false);
	await inside
		.getByRole('button', { name: /fullscreen/i })
		.first()
		.click();
	await expect
		.poll(() => page.evaluate(() => !!document.fullscreenElement))
		.toBe(true);
	await page.evaluate(() => document.exitFullscreen());
	await expect
		.poll(() =>
			inside
				.locator('video')
				.evaluate((node) => (node as HTMLVideoElement).paused)
		)
		.toBe(false);
	await frame.evaluate((node) => {
		node.style.position = 'fixed';
		node.style.top = '-3000px';
	});
	await expect
		.poll(() =>
			inside
				.locator('video')
				.evaluate((node) => (node as HTMLVideoElement).paused)
		)
		.toBe(true);
});

test('reported Bunny errors switch to the standard player', async ({
	page
}) => {
	await hostedVideo(page);
	await page.route('https://player.mediadelivery.net/**', (route) =>
		route.fulfill({
			contentType: 'text/html',
			body: playerFixture.replace(
				'ready();\n </script>',
				"ready(); setTimeout(()=>emit('error',{}),100);\n </script>"
			)
		})
	);
	await page.goto('/feed');
	await expect(page.locator('.video-player video')).toHaveCount(1);
	await expect(page.locator('.bunny-video iframe')).toHaveCount(0);
});

for (const surface of ['quote', 'saved', 'composer']) {
	test(`${surface} video previews show a thumbnail and badge without loading a player`, async ({
		page
	}) => {
		const video = {
			cid: 'bafy-video',
			mime: 'video/mp4',
			size: 100,
			alt: 'Knitting a sleeve',
			aspectRatio: { width: 9, height: 16 },
			thumbnail: 'https://video-fixture.craftsky.test/poster.png',
			playlist: 'https://video-fixture.craftsky.test/playlist.m3u8'
		};
		let target: any;
		await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
			const response = await route.fetch(),
				body = await response.json();
			const outer = body.items[0].post;
			target = {
				...outer,
				uri: 'at://did:plc:alice/social.craftsky.feed.post/quoted-video',
				rkey: 'quoted-video',
				text: 'A video of knitting',
				images: [],
				video,
				quote: null,
				quoteView: null,
				viewerHasSaved: true
			};
			outer.images = [];
			delete outer.video;
			outer.quote = { uri: target.uri, cid: target.cid };
			outer.quoteView = { state: 'visible', post: target };
			await route.fulfill({ response, json: body });
		});
		await page.route(
			'https://video-fixture.craftsky.test/poster.png',
			(route) =>
				route.fulfill({
					path: resolve('static/app-icon.png'),
					contentType: 'image/png'
				})
		);
		await page.goto('/feed');
		await expect(page.locator('.quote-card .video-thumbnail')).toBeVisible();
		const expected = '/posts/did%3Aplc%3Aalice/quoted-video';
		if (surface === 'saved') {
			await page.route('**/api/appview/v1/saved-posts*', (route) =>
				route.fulfill({
					json: {
						items: [
							{ post: target, savedAt: '2026-11-06T12:00:00Z', folderId: null }
						]
					}
				})
			);
			await page.goto('/saved?all=true');
		} else if (surface === 'composer') {
			await page.route('**/api/appview/v1/posts/**', (route) =>
				route.fulfill({ json: target })
			);
			await page.goto('/compose/quote?did=did%3Aplc%3Aalice&rkey=quoted-video');
		}
		const preview = page.getByRole('img', {
			name: 'Video: Knitting a sleeve',
			exact: true
		});
		await expect(preview).toBeVisible();
		await expect(preview.locator('img')).toHaveAttribute(
			'src',
			video.thumbnail
		);
		await expect(preview.locator('svg')).toBeVisible();
		await expect(preview).toHaveCSS('aspect-ratio', '16 / 9');
		await expect(preview.locator('img')).toHaveCSS('object-fit', 'cover');
		await expect(page.locator('video,iframe')).toHaveCount(0);
		if (surface !== 'composer')
			await expect(preview.locator('..')).toHaveAttribute('href', expected);
	});
}

test('oversized video selection explains the 300 MB limit and retains composer text', async ({
	page
}) => {
	await page.addInitScript(() => {
		const descriptor = Object.getOwnPropertyDescriptor(Blob.prototype, 'size')!;
		Object.defineProperty(File.prototype, 'size', {
			get() {
				return this.name === 'oversized.mov'
					? 300000001
					: descriptor.get!.call(this);
			}
		});
	});
	await page.goto('/compose');
	await expect(
		page.getByRole('heading', { name: 'Media', exact: true })
	).toBeVisible();
	await expect(page.getByText('0 described', { exact: true })).toHaveCount(0);
	await expect(page.locator('.photo-picker small')).toHaveText(
		'Choose up to 8 photos or a video.'
	);
	const text = page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
	await text.fill('Keep this draft');
	await page.getByLabel('Choose a video', { exact: true }).setInputFiles({
		name: 'oversized.mov',
		mimeType: 'video/quicktime',
		buffer: Buffer.from([0, 0, 0, 20, ...Buffer.from('ftypqt  ')])
	});
	await expect(
		page.getByText(
			'This video is too large. Videos must be 300 MB or smaller. Choose a smaller file or export a smaller copy.',
			{ exact: true }
		)
	).toBeVisible();
	await expect(text).toHaveValue('Keep this draft');
	await expect(page.getByText('Selected video', { exact: true })).toHaveCount(
		0
	);
});

for (const width of [1280, 820]) {
	test(`selected video opens local playback details and retains its description on return at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.route('https://video.bunnycdn.com/**', () => {
			throw new Error('Inspection must not upload video');
		});
		await page.goto('/compose');
		await page.locator('.media-picker summary').click();
		await expect(
			page.getByText('Up to 8 photos · 40 MB each', { exact: true })
		).toBeVisible();
		await expect(
			page.getByText('MP4 or MOV · up to 10 minutes · 300 MB', { exact: true })
		).toBeVisible();
		const photoChooserPromise = page.waitForEvent('filechooser');
		await page.getByRole('button', { name: /^Choose photos/ }).click();
		const photoChooser = await photoChooserPromise;
		expect(photoChooser.isMultiple()).toBe(true);
		expect(await photoChooser.element().getAttribute('accept')).toBe(
			'image/jpeg,image/png,image/webp'
		);
		await page.locator('.media-picker summary').click();
		const videoChooserPromise = page.waitForEvent('filechooser');
		await page.getByRole('button', { name: /^Choose a video/ }).click();
		const videoChooser = await videoChooserPromise;
		expect(videoChooser.isMultiple()).toBe(false);
		expect(await videoChooser.element().getAttribute('accept')).toBe(
			'video/mp4,video/quicktime,.mp4,.mov'
		);
		await videoChooser.setFiles(
			resolve('tests/fixtures/video/local-preview.mp4')
		);
		await expect(
			page.getByRole('heading', { name: 'Video details', exact: true })
		).toBeVisible();
		const video = page.getByLabel('Local video preview', { exact: true });
		await expect(video).toHaveAttribute('src', /^blob:/);
		await video.evaluate((node) => (node as HTMLVideoElement).play());
		await expect
			.poll(() =>
				video.evaluate((node) => (node as HTMLVideoElement).currentTime)
			)
			.toBeGreaterThan(0);
		const element = await video.elementHandle();
		await page
			.getByLabel('Video description', { exact: true })
			.fill('Hands knitting a blue scarf');
		await page
			.getByRole('button', { name: 'Back to post', exact: true })
			.click();
		await expect(
			page.getByRole('button', { name: 'Inspect selected video', exact: true })
		).toBeVisible();
		await expect(page.locator('.video-attachment')).toContainText(
			'Selected video'
		);
		await expect(page.locator('.video-attachment')).toContainText(
			'Video details'
		);
		await expect(
			page
				.locator('.video-attachment')
				.getByRole('button', { name: 'Remove video', exact: true })
		).toBeVisible();
		expect(
			await element!.evaluate((node) => (node as HTMLVideoElement).paused)
		).toBe(true);
		await page
			.getByRole('button', { name: 'Inspect selected video', exact: true })
			.click();
		await expect(
			page.getByLabel('Video description', { exact: true })
		).toHaveValue('Hands knitting a blue scarf');
		await page.keyboard.press('Escape');
		await expect(
			page.getByRole('heading', { name: 'Video details', exact: true })
		).toHaveCount(0);
		await expect(
			page.getByRole('button', { name: 'Inspect selected video', exact: true })
		).toBeFocused();
		await expect(page.getByLabel('Replace video', { exact: true })).toHaveCount(
			0
		);
		await page
			.getByRole('button', { name: 'Remove video', exact: true })
			.click();
		await page
			.getByLabel('Choose a video', { exact: true })
			.setInputFiles(resolve('tests/fixtures/video/local-preview.mp4'));
		await expect(
			page.getByRole('heading', { name: 'Video details', exact: true })
		).toBeVisible();
	});
}
