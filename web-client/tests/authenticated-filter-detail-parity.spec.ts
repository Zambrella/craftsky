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
test('native compact filter form uses search, dependent choices and Cancel without applying edits', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/app/home?craft=knitting');
	const open = () =>
		page.getByRole('button', { name: /^Filter projects/ }).click();
	await open();
	const dialog = page.getByRole('dialog', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await expect(dialog).toBeVisible();
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
	const subtype = dialog.getByRole('searchbox', {
		name: 'Project subtype',
		exact: true
	});
	await expect(subtype).toBeDisabled();
	await expect(type).toHaveCSS('border-top-width', '0px');
	await expect(type).toHaveCSS('box-shadow', 'none');
	await type.fill('gar');
	const choices = dialog.getByRole('group', {
		name: 'Project type options',
		exact: true
	});
	await expect(choices.getByRole('checkbox')).toHaveCount(1);
	await page.screenshot({
		path: test.info().outputPath('native-filter-picker.png')
	});
	await choices.getByRole('checkbox', { name: 'Garment', exact: true }).check();
	await expect(type).toHaveValue('');
	await expect(subtype).toBeEnabled();
	await subtype.fill('sweater');
	await expect(choices).toHaveCount(0);
	await dialog.getByRole('checkbox', { name: 'Sweater', exact: true }).check();
	await subtype.press('Escape');
	await expect(dialog).toBeVisible();
	await dialog
		.getByRole('button', {
			name: 'Remove Garment from Project type',
			exact: true
		})
		.click();
	await expect(subtype).toBeDisabled();
	await expect(
		dialog.getByRole('button', {
			name: 'Remove Sweater from Project subtype',
			exact: true
		})
	).toHaveCount(0);
	const color = dialog.getByRole('searchbox', { name: 'Color', exact: true });
	await color.fill('blue');
	await dialog.getByRole('checkbox', { name: 'Blue', exact: true }).check();
	await color.press('Enter');
	await expect(dialog).toBeVisible();
	expect(new URL(page.url()).searchParams.has('color')).toBe(false);
	await dialog
		.getByRole('button', { name: 'Close filters', exact: true })
		.click();
	await open();
	await expect(
		dialog.getByRole('button', { name: 'Remove Blue from Color', exact: true })
	).toHaveCount(0);
	await expect(
		dialog.getByRole('searchbox', { name: 'Project type', exact: true })
	).toHaveValue('');
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
});
test('native selection limit and Clear all retain craft/sort until explicit Apply', async ({
	page
}) => {
	await page.goto(
		'/app/home?craft=knitting&sort=popular&color=blue&selfDrafted=true'
	);
	await page.getByRole('button', { name: /^Filter projects/ }).click();
	const dialog = page.getByRole('dialog', {
		name: 'Filter Knitting projects',
		exact: true
	});
	const color = dialog.getByRole('searchbox', { name: 'Color', exact: true });
	await color.click();
	const choices = dialog.getByRole('group', {
		name: 'Color options',
		exact: true
	});
	for (let i = 0; i < 9; i++)
		await choices.locator('input:not(:checked):not(:disabled)').first().check();
	await expect(choices.locator('input:checked')).toHaveCount(10);
	await expect(
		choices.locator('input:not(:checked):not(:disabled)')
	).toHaveCount(0);
	await color.press('Escape');
	await dialog.getByRole('button', { name: 'Clear all', exact: true }).click();
	await expect(
		dialog.getByRole('switch', { name: 'Self drafted', exact: true })
	).not.toBeChecked();
	expect(new URL(page.url()).searchParams.get('color')).toBe('blue');
	await dialog
		.getByRole('button', { name: 'Apply filters', exact: true })
		.click();
	await expect(dialog).not.toBeVisible();
	const query = new URL(page.url()).searchParams;
	expect(query.get('craft')).toBe('knitting');
	expect(query.get('sort')).toBe('popular');
	expect(query.has('color')).toBe(false);
	expect(query.has('selfDrafted')).toBe(false);
});
