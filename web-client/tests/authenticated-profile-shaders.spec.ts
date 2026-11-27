import { expect, test } from '@playwright/test';

// Compile and read real shader pixels on hosts without a hardware GPU.
test.use({
	launchOptions: {
		args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader']
	}
});

test('translucent shader pigments follow profile colours in WebGL', async ({
	context,
	page
}) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('**/api/service-status', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	const account = {
		id: '0c10030a-6208-43c2-8048-d3edcc609271',
		did: 'did:plc:alice',
		handle: 'alice.test',
		token: Buffer.alloc(32, 1).toString('base64url')
	};
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
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.goto('/settings/customisation');
	const preview = page.getByLabel('Profile customisation preview');
	for (const background of [
		'Aurora',
		'Colour pools',
		'Sunset',
		'Pigment',
		'Marble',
		'Woven'
	]) {
		await page.getByText(background, { exact: true }).click();
		const samples: number[][] = [];
		for (const colour of ['Cobalt', 'Seafoam', 'Terracotta', 'Ink']) {
			await page.getByText(colour, { exact: true }).click();
			const canvas = preview.locator('canvas');
			await expect(canvas).toBeVisible();
			const pixels = await canvas.evaluate((element: HTMLCanvasElement) => {
				const gl = element.getContext('webgl')!;
				// Redraw the still immediately before readback. These are shader
				// pixels independently of the CSS profile base colour.
				gl.drawArrays(gl.TRIANGLES, 0, 6);
				const sample = new Uint8Array(4);
				const result: number[] = [];
				for (const x of [0.2, 0.5, 0.8])
					for (const y of [0.2, 0.5, 0.8]) {
						gl.readPixels(
							Math.floor(element.width * x),
							Math.floor(element.height * y),
							1,
							1,
							gl.RGBA,
							gl.UNSIGNED_BYTE,
							sample
						);
						result.push(...sample);
					}
				if (gl.getError() !== gl.NO_ERROR)
					throw new Error('WebGL readback failed');
				return result;
			});
			expect(pixels.some((value, index) => index % 4 === 3 && value > 0)).toBe(
				true
			);
			samples.push(pixels);
		}
		for (const sample of samples.slice(1))
			expect(sample, background).not.toEqual(samples[0]);
		const ink = samples.at(-1)!;
		for (let index = 0; index < ink.length; index += 4) {
			expect(
				Math.abs(ink[index] - ink[index + 1]),
				background
			).toBeLessThanOrEqual(1);
			expect(
				Math.abs(ink[index + 1] - ink[index + 2]),
				background
			).toBeLessThanOrEqual(1);
		}
	}
});
