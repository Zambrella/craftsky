import { test, expect } from '@playwright/test';
const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'healthy' } })
	);
	await context.addCookies([
		{
			name: 'craftsky_account_' + alice.id,
			value: Buffer.from(JSON.stringify(alice)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: alice.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.route('**/api/appview/v1/projects*', (route) =>
		route.fulfill({ json: { items: [] } })
	);
});
test('compact filters show a flat list with searchable nested project types', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/feed?craft=knitting');
	const open = () =>
		page.getByRole('button', { name: /^Filter projects/ }).click();
	await open();
	const dialog = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await expect(dialog).toBeVisible();
	const fields = dialog.locator('.filter-fields > [data-filter]');
	await expect(fields).toHaveCount(5);
	expect(
		await fields.evaluateAll((controls) =>
			controls.map((control) => control.getAttribute('data-filter'))
		)
	).toEqual([
		'status',
		'patternDifficulty',
		'selfDrafted',
		'projectItem',
		'knittingMethod'
	]);
	await expect(dialog.getByText('Status', { exact: true })).toHaveCount(1);
	await expect(
		dialog.getByRole('searchbox', { name: 'Status', exact: true })
	).toBeVisible();
	await expect(dialog.locator('summary')).toHaveCount(0);
	for (const family of ['patternDifficulty', 'projectItem']) {
		await expect(dialog.locator(`[data-filter="${family}"]`)).toBeVisible();
	}
	expect(
		await dialog.evaluate((x) => ({
			top: x.getBoundingClientRect().top,
			height: x.clientHeight,
			width: x.clientWidth
		}))
	).toEqual({ top: 0, height: 874, width: 402 });
	const type = dialog.getByRole('searchbox', {
		name: 'Project type',
		exact: true
	});
	await expect(
		dialog.getByRole('searchbox', { name: 'Project subtype', exact: true })
	).toHaveCount(0);
	await expect(type).toHaveCSS('border-top-width', '0px');
	await expect(type).toHaveCSS('box-shadow', 'none');
	const choices = dialog.getByRole('group', {
		name: 'Project type options',
		exact: true
	});
	await type.click();
	const rows = choices.getByRole('checkbox');
	const garment = choices.getByRole('checkbox', {
		name: 'Garment',
		exact: true
	});
	await expect(rows.last()).toHaveAccessibleName('Other');
	await expect(garment.locator('..')).toHaveCSS('margin-block-start', '6px');
	await type.fill('garment');
	await expect(rows.first()).toHaveAccessibleName('Garment');
	await expect(rows.first().locator('..')).toHaveCSS(
		'margin-block-start',
		'0px'
	);
	await expect(
		choices.getByRole('checkbox', { name: 'Jumper', exact: true })
	).toBeVisible();
	await page.screenshot({
		path: test.info().outputPath('native-filter-picker.png')
	});
	// Click the option label, as a person does, rather than the checkbox itself.
	await choices.getByText('Garment', { exact: true }).click();
	await expect(choices).toHaveCount(0);
	await expect(type).toBeFocused();
	await expect(type).toHaveValue('');
	await type.fill('pullover');
	await expect(rows).toHaveCount(2);
	await expect(rows.first()).toHaveAccessibleName('Garment');
	const jumperOption = rows.last();
	await expect(jumperOption).toHaveAccessibleName('Jumper');
	await expect(jumperOption).toHaveAccessibleDescription(
		'Garment · Jumper Pullover'
	);
	expect((await jumperOption.boundingBox())!.x).toBeGreaterThan(
		(await rows.first().boundingBox())!.x
	);
	await expect(rows.first().locator('..')).toHaveCSS('font-weight', '700');
	await expect(jumperOption.locator('..')).toHaveCSS('font-weight', '400');
	await expect(jumperOption.locator('..')).toHaveCSS(
		'margin-block-start',
		'0px'
	);
	await type.press('Enter');
	const jumper = dialog.getByRole('button', {
		name: 'Remove Garment · Jumper from Project type',
		exact: true
	});
	await expect(jumper).toBeVisible();
	await expect(dialog).toBeVisible();
	await dialog
		.getByRole('button', {
			name: 'Remove Garment from Project type',
			exact: true
		})
		.click();
	await expect(jumper).toBeVisible();
	await jumper.click();
	await expect(jumper).toHaveCount(0);
	const method = dialog.getByRole('searchbox', {
		name: 'Knitting method',
		exact: true
	});
	const methods = dialog.getByRole('group', {
		name: 'Knitting method options',
		exact: true
	});
	await method.click();
	const labels = await methods
		.getByRole('checkbox')
		.evaluateAll((options) =>
			options.map((option) => option.getAttribute('aria-label'))
		);
	expect(labels).toEqual(['Hand', 'Loom', 'Machine']);
	await method.fill('Loom knitting');
	await expect(methods).toHaveText('Loom');
	await method.fill('Machine knitting');
	await expect(methods.getByRole('checkbox')).toHaveCount(1);
	await expect(
		methods.getByRole('checkbox', { name: 'Machine' })
	).toBeVisible();
	await expect(methods).toHaveText('Machine');
	await methods.getByRole('checkbox', { name: 'Machine', exact: true }).click();
	await expect(dialog).toBeVisible();
	expect(new URL(page.url()).searchParams.has('knittingMethod')).toBe(false);
	await dialog
		.getByRole('button', { name: 'Back to projects', exact: true })
		.click();
	await open();
	await expect(type).toHaveValue('');
	await expect(
		dialog.getByRole('button', { name: 'Remove Machine from Knitting method' })
	).toHaveCount(0);
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
});
test('embroidery styles can be selected independently and historical selections can be cleared', async ({
	page
}) => {
	const style = 'social.craftsky.project.embroidery.defs#sashiko';
	await page.goto(
		`/feed?craft=embroidery&embroideryStyle=${encodeURIComponent(style)}`
	);
	await page.getByRole('button', { name: /^Filter projects/ }).click();
	const dialog = page.getByRole('region', {
		name: 'Filter Embroidery projects',
		exact: true
	});
	const styleInput = dialog.getByRole('searchbox', {
		name: 'Embroidery style',
		exact: true
	});
	await expect(styleInput).toBeEnabled();
	await expect(
		dialog.getByRole('searchbox', { name: 'Project type', exact: true })
	).toBeEnabled();
	await dialog
		.getByRole('button', { name: 'Remove Sashiko from Embroidery style' })
		.click();
	await styleInput.fill('Needlepoint');
	await dialog
		.getByRole('checkbox', { name: 'Needlepoint', exact: true })
		.click();
	await dialog.getByRole('button', { name: 'Apply filters' }).click();
	await expect(dialog).toBeHidden();
	await expect(page).toHaveURL(
		(url) =>
			url.searchParams.get('embroideryStyle') ===
			'social.craftsky.project.embroidery.defs#needlepoint'
	);
	const query = new URL(page.url()).searchParams;
	expect(query.getAll('embroideryStyle')).toEqual([
		'social.craftsky.project.embroidery.defs#needlepoint'
	]);
	expect(query.has('embroideryMethod')).toBe(false);
});
test('native selection limit and Clear all apply cleared filters and return to the craft feed', async ({
	page
}) => {
	await page.goto('/feed?craft=knitting&sort=popular&selfDrafted=true');
	await page.getByRole('button', { name: /^Filter projects/ }).click();
	const dialog = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await expect(
		dialog.getByRole('switch', { name: 'Self drafted' })
	).toBeChecked();
	const type = dialog.getByRole('searchbox', {
		name: 'Project type',
		exact: true
	});
	const choices = dialog.getByRole('group', {
		name: 'Project type options',
		exact: true
	});
	for (let i = 0; i < 10; i++) {
		await type.click();
		await choices.locator('input:not(:checked):not(:disabled)').first().click();
	}
	await type.click();
	await expect(choices.locator('input:checked')).toHaveCount(10);
	await expect(
		choices.locator('input:not(:checked):not(:disabled)')
	).toHaveCount(0);
	await type.press('Escape');
	await dialog.getByRole('button', { name: 'Clear all', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await expect(page).toHaveURL('/feed?craft=knitting&sort=popular');
});

test('project item filters show alternative names in the dropdown', async ({
	page
}) => {
	await page.goto('/feed?craft=sewing');
	await page
		.getByRole('button', { name: 'Filter projects', exact: true })
		.click();
	const filters = page.getByRole('region', {
		name: 'Filter Sewing projects',
		exact: true
	});
	const type = filters.getByRole('searchbox', {
		name: 'Project type',
		exact: true
	});
	await type.fill('costume');
	await expect(
		filters
			.getByRole('checkbox', { name: 'Costume', exact: true })
			.locator('..')
	).toHaveCSS('font-weight', '700');
	await type.fill('home decor');
	const homeGoods = filters.getByRole('checkbox', {
		name: 'Home goods',
		exact: true
	});
	await expect(homeGoods).toHaveAccessibleDescription('Home decor');
	await type.fill('overalls');
	const dungarees = filters.getByRole('checkbox', {
		name: 'Dungarees',
		exact: true
	});
	await expect(dungarees).toHaveAccessibleDescription(
		'Garment · Dungarees Overalls'
	);
	await expect(filters.getByText('Overalls', { exact: true })).toBeVisible();
	await dungarees.click();
	await expect(
		filters.getByRole('button', {
			name: 'Remove Garment · Dungarees from Project type'
		})
	).toHaveText('Garment · Dungarees');
	await type.fill('mending');
	await expect(
		filters.getByRole('checkbox', { name: 'Repair', exact: true })
	).toHaveAccessibleDescription('Alteration and repair · Repair Mending');
});

test('Sewing item filters apply, reopen and clear through the projects request', async ({
	page
}) => {
	await page.goto(
		'/feed?craft=sewing&settUnit=ends-per-inch&technique=frenchSeams'
	);
	const open = () =>
		page.getByRole('button', { name: 'Filter projects', exact: true }).click();
	await expect(page.getByLabel('Filters applied', { exact: true })).toHaveCount(
		0
	);
	await open();
	const filters = page.getByRole('region', {
		name: 'Filter Sewing projects',
		exact: true
	});
	await expect(
		filters.getByRole('searchbox', { name: 'Techniques', exact: true })
	).toHaveCount(0);
	const type = filters.getByRole('searchbox', {
		name: 'Project type',
		exact: true
	});
	await type.fill('Garment');
	await filters.getByRole('checkbox', { name: 'Garment', exact: true }).click();
	await type.fill('Tote bag');
	await filters
		.getByRole('checkbox', { name: 'Tote bag', exact: true })
		.click();
	const request = page.waitForRequest((request) => {
		const url = new URL(request.url());
		return (
			url.pathname === '/api/appview/v1/projects' &&
			url.searchParams.has('projectItem')
		);
	});
	await filters
		.getByRole('button', { name: 'Apply filters', exact: true })
		.click();
	const query = new URL((await request).url()).searchParams;
	const items = [
		'social.craftsky.project.defs#garment',
		'social.craftsky.project.sewing.defs#tote'
	];
	expect(query.getAll('projectItem').sort()).toEqual(items);
	expect(query.has('projectType')).toBe(false);
	expect(query.has('projectSubtype')).toBe(false);
	expect(query.has('settUnit')).toBe(false);
	expect(query.has('technique')).toBe(false);
	await expect(
		page.getByLabel('Filters applied', { exact: true })
	).toBeVisible();
	expect(new URL(page.url()).searchParams.getAll('projectItem').sort()).toEqual(
		items
	);
	await open();
	await expect(
		filters.getByRole('button', {
			name: 'Remove Garment from Project type',
			exact: true
		})
	).toBeVisible();
	await expect(
		filters.getByRole('button', {
			name: 'Remove Accessory · Tote bag from Project type',
			exact: true
		})
	).toBeVisible();
	await filters.getByRole('button', { name: 'Clear all', exact: true }).click();
	await expect(filters).not.toBeVisible();
	await expect
		.poll(() => new URL(page.url()).searchParams.has('projectItem'))
		.toBe(false);
});

test('legacy type and subtype filters narrow the results until the item selection is edited', async ({
	page
}) => {
	const typeId = 'social.craftsky.project.defs#garment';
	const subtypeId = 'social.craftsky.project.knitting.defs#pullover';
	const accessoryId = 'social.craftsky.project.defs#accessory';
	const requests: URLSearchParams[] = [];
	await page.route('**/api/appview/v1/projects*', (route) => {
		requests.push(new URL(route.request().url()).searchParams);
		return route.fulfill({ json: { items: [] } });
	});
	const legacyQuery = new URLSearchParams({
		craft: 'knitting',
		projectType: typeId,
		projectSubtype: subtypeId
	});
	await page.goto(`/feed?${legacyQuery}`);
	const open = () =>
		page.getByRole('button', { name: /^Filter projects/ }).click();
	const dialog = page.getByRole('region', { name: 'Filter Knitting projects' });
	const apply = () =>
		dialog.getByRole('button', { name: 'Apply filters' }).click();
	await open();
	await expect(
		dialog.getByRole('button', {
			name: 'Remove Garment · Jumper from Project type',
			exact: true
		})
	).toBeVisible();
	await expect(
		dialog.getByRole('button', {
			name: 'Remove Garment from Project type',
			exact: true
		})
	).toHaveCount(0);
	await dialog.getByRole('switch', { name: 'Self drafted' }).check();
	await apply();
	await expect(page).toHaveURL(
		({ searchParams }) =>
			searchParams.get('selfDrafted') === 'true' &&
			searchParams.get('projectType') === typeId &&
			searchParams.get('projectSubtype') === subtypeId &&
			!searchParams.has('projectItem')
	);
	await expect.poll(() => requests.at(-1)?.get('selfDrafted')).toBe('true');
	const legacy = requests.at(-1)!;
	expect(legacy.get('projectType')).toBe(typeId);
	expect(legacy.get('projectSubtype')).toBe(subtypeId);
	expect(legacy.has('projectItem')).toBe(false);

	await open();
	await dialog
		.getByRole('searchbox', { name: 'Project type', exact: true })
		.fill('Accessory');
	await dialog
		.getByRole('checkbox', { name: 'Accessory', exact: true })
		.click();
	await apply();
	await expect(page).toHaveURL((url) => url.searchParams.has('projectItem'));
	await expect
		.poll(() => requests.at(-1)?.getAll('projectItem').sort())
		.toEqual([accessoryId, subtypeId]);
	expect(requests.at(-1)!.has('projectType')).toBe(false);
	expect(requests.at(-1)!.has('projectSubtype')).toBe(false);
});

test('native sort sheets preserve craft filters and move to a pointer menu on wide layouts', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/feed?craft=knitting&status=finished');
	await page.getByLabel('Sort projects: Newest', { exact: true }).click();
	const menu = page.locator('.choice-menu[open] .menu-items');
	await expect(menu).toHaveCSS('position', 'fixed');
	await menu.getByRole('link', { name: 'Popular', exact: true }).click();
	await expect(page).toHaveURL(/sort=popular/);
	expect(new URL(page.url()).searchParams.get('craft')).toBe('knitting');
	expect(new URL(page.url()).searchParams.get('status')).toBe('finished');
	await page.setViewportSize({ width: 1280, height: 900 });
	await page.getByLabel('Sort projects: Popular', { exact: true }).click();
	await expect(menu).toHaveCSS('border-top-left-radius', '14px');
	expect(
		await menu.evaluate((el) => el.getBoundingClientRect().width)
	).toBeLessThanOrEqual(280);
	await expect(page.locator('html')).not.toHaveCSS('overflow', 'hidden');
	await page.keyboard.press('Escape');
	await expect(page.locator('.choice-menu[open]')).toHaveCount(0);
});

test('project filters are a content-column route with Back and Forward and no modal history after Apply', async ({
	page
}) => {
	await page.goto('/feed');
	await page
		.getByRole('navigation', { name: 'Home feeds' })
		.getByRole('link', { name: 'Knitting', exact: true })
		.click();
	await expect(page).toHaveURL(/craft=.*knitting/);
	const origin = page.url();
	// Wait for route navigation, which can outlast the default assertion timeout
	// when the local Worker is serving the filter route's server data.
	await Promise.all([
		page.waitForURL(/\/feed\/filters\?/),
		page.getByRole('button', { name: 'Filter projects', exact: true }).click()
	]);
	const filters = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await expect(filters).toBeVisible();
	await expect(page.locator('dialog[open]')).toHaveCount(0);
	const main = (await page.locator('.app-main').boundingBox())!;
	const bounds = (await filters.boundingBox())!;
	expect(Math.abs(main.x - bounds.x)).toBeLessThan(2);
	expect(Math.abs(main.width - bounds.width)).toBeLessThan(2);
	await filters
		.getByRole('switch', { name: 'Self drafted', exact: true })
		.check();
	await page.goBack();
	await expect(page).toHaveURL(origin);
	await page.goForward();
	await expect(filters).toBeVisible();
	await filters
		.getByRole('switch', { name: 'Self drafted', exact: true })
		.check();
	await filters
		.getByRole('button', { name: 'Apply filters', exact: true })
		.click();
	await expect(page).toHaveURL(/selfDrafted=true/);
	await expect(filters).toHaveCount(0);
	await page.goBack();
	await expect(page).toHaveURL(/\/feed$/);
});
