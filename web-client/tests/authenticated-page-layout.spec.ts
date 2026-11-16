import { test, expect, type Locator } from '@playwright/test';

const fixture = 'http://127.0.0.1:19847';
const alice = {
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
			name: 'craftsky_account_' + alice.id,
			value: Buffer.from(JSON.stringify(alice)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: alice.id,
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
});

async function appBar(locator: Locator) {
	return locator.evaluate((node) => {
		const title = node.matches('h1,h2') ? node : node.querySelector('h1,h2')!;
		const box = node.getBoundingClientRect(),
			barStyle = getComputedStyle(node),
			titleStyle = getComputedStyle(title);
		return {
			x: box.x,
			width: box.width,
			height: box.height,
			border: barStyle.borderBottom,
			background: barStyle.backgroundColor,
			font: titleStyle.fontFamily,
			size: titleStyle.fontSize,
			weight: titleStyle.fontWeight,
			lineHeight: titleStyle.lineHeight,
			letterSpacing: titleStyle.letterSpacing
		};
	});
}

for (const width of [390, 1280, 1767]) {
	test(`pages and composers share the main column and app bar at ${width}px`, async ({
		page
	}, testInfo) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/feed');
		const feedBar = page.locator('.feed-content > .workspace-heading');
		await expect(feedBar).toBeVisible();
		const standard = await appBar(feedBar);
		expect(standard.height).toBe(56);
		expect(standard.size).toBe('20px');
		expect(standard.weight).toBe('600');
		for (const [path, selector, workspace] of [
			[
				'/settings',
				'.settings-workspace > .workspace-heading',
				'.settings-workspace'
			],
			[
				'/settings/privacy',
				'.settings-workspace > .workspace-heading',
				'.settings-workspace'
			],
			['/settings/business', '.business-details-header', '.settings-workspace'],
			[
				'/settings/products',
				'.settings-workspace > .workspace-heading',
				'.settings-workspace'
			],
			[
				'/settings/events',
				'.settings-workspace > .workspace-heading',
				'.settings-workspace'
			],
			['/compose', '.composer-heading', '.composer'],
			['/compose/project', '.composer-heading', '.composer'],
			['/compose/event', '.business-composer-heading', '.event-composer'],
			['/compose/product', '.business-composer-heading', '.product-composer'],
			['/profile/edit', '.workspace-header', ''],
			['/drafts', '.draft-appbar', '.draft-workspace'],
			['/scheduled', '.draft-appbar', '.draft-workspace'],
			['/messages', '.message-header', '.messages-workspace']
		]) {
			await test.step(path, async () => {
				await page.goto(path);
				const header = page.locator(selector).first();
				await expect(header).toBeVisible();
				const main = await page.locator('main.app-main').boundingBox();
				const measured = await appBar(header);
				expect(measured.x, path).toBeCloseTo(main!.x, 0);
				expect(measured.width, path).toBeCloseTo(main!.width, 0);
				const { x: _x, width: _width, ...appearance } = measured;
				const { x: _standardX, width: _standardWidth, ...expected } = standard;
				expect(appearance, path).toEqual(expected);
				if (workspace) {
					const box = await page.locator(workspace).first().boundingBox();
					expect(box!.x, path).toBeCloseTo(main!.x, 0);
					expect(box!.width, path).toBeCloseTo(main!.width, 0);
				}
				if (
					['/settings', '/compose/product', '/compose/project'].includes(path)
				)
					await page.screenshot({
						path: testInfo.outputPath(
							`${path.slice(1).replaceAll('/', '-')}-${width}.png`
						)
					});
				if (path.startsWith('/settings/')) {
					await page
						.getByRole('link', { name: 'Back to Settings', exact: true })
						.click();
					await expect(page).toHaveURL(/\/settings$/);
				}
			});
		}
		// Opening photo details must keep the same bar and column as its parent.
		await page.goto('/compose/product');
		const png = await page.evaluate(() => {
			const canvas = document.createElement('canvas');
			canvas.width = 40;
			canvas.height = 30;
			return canvas.toDataURL('image/png').split(',')[1];
		});
		await page.getByLabel('Add images', { exact: true }).setInputFiles({
			name: 'product.png',
			mimeType: 'image/png',
			buffer: Buffer.from(png, 'base64')
		});
		const photoBar = page.locator('.photo-page-header');
		await expect(photoBar).toBeVisible();
		const measured = await appBar(photoBar);
		const main = await page.locator('main.app-main').boundingBox();
		expect(measured.x).toBeCloseTo(main!.x, 0);
		expect(measured.width).toBeCloseTo(main!.width, 0);
		const { x: _x, width: _width, ...appearance } = measured;
		const { x: _standardX, width: _standardWidth, ...expected } = standard;
		expect(appearance).toEqual(expected);
	});
}

for (const width of [390, 1767])
	test(`project and photo detail headers share the standard title at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/feed');
		const standard = await appBar(
			page.locator('.feed-content > .workspace-heading')
		);
		const typography = ({
			height,
			font,
			size,
			weight,
			lineHeight,
			letterSpacing
		}: Awaited<ReturnType<typeof appBar>>) => ({
			height,
			font,
			size,
			weight,
			lineHeight,
			letterSpacing
		});
		await page.goto('/compose/project');
		await page.getByRole('button', { name: /Pattern details/ }).click();
		const projectBar = page.locator('.project-detail-heading');
		await expect(projectBar).toBeVisible();
		expect(typography(await appBar(projectBar))).toEqual(typography(standard));
		await page
			.getByRole('button', { name: 'Back to project', exact: true })
			.click();
		const png = await page.evaluate(() => {
			const canvas = document.createElement('canvas');
			canvas.width = 40;
			canvas.height = 30;
			return canvas.toDataURL('image/png').split(',')[1];
		});
		await page.getByLabel('Add images', { exact: true }).setInputFiles({
			name: 'project.png',
			mimeType: 'image/png',
			buffer: Buffer.from(png, 'base64')
		});
		const photoBar = page.locator('.photo-page-header');
		await expect(photoBar).toBeVisible();
		expect(typography(await appBar(photoBar))).toEqual(typography(standard));
		await page.getByRole('button', { name: 'Edit photo', exact: true }).click();
		const editorBar = page.locator('.image-edit-heading');
		await expect(editorBar).toBeVisible();
		expect(typography(await appBar(editorBar))).toEqual(typography(standard));
	});
