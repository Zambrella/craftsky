import { test, expect } from '@playwright/test';

for (const colorScheme of ['light', 'dark'] as const) {
	for (const reducedMotion of ['no-preference', 'reduce'] as const) {
		test(`floating filter keeps primary interaction colors in ${colorScheme} with ${reducedMotion} motion`, async ({
			page
		}) => {
			await page.setViewportSize({ width: 800, height: 900 });
			await page.emulateMedia({ colorScheme, reducedMotion });
			await page.goto('/feed?craft=sewing');
			await page.getByRole('button', { name: 'Explore CraftSky' }).click();
			const button = page.getByRole('button', { name: 'Filter projects' });
			await expect(button).toBeVisible();
			const colors = await button.evaluate((node) => {
				const sample = document.createElement('span');
				node.append(sample);
				const resolve = (variable: string) => {
					sample.style.backgroundColor = `var(${variable})`;
					return getComputedStyle(sample).backgroundColor;
				};
				const result = {
					hover: resolve('--primary-hover'),
					pressed: resolve('--primary-pressed'),
					text: getComputedStyle(node).color
				};
				sample.remove();
				return result;
			});
			await button.hover();
			await expect(button).toHaveCSS('background-color', colors.hover);
			await expect(button).toHaveCSS('color', colors.text);
			await expect(button).toHaveCSS(
				'transform',
				reducedMotion === 'reduce' ? 'none' : 'matrix(1, 0, 0, 1, -1, -1)'
			);
			await page.mouse.down();
			await expect(button).toHaveCSS('background-color', colors.pressed);
			await expect(button).toHaveCSS('box-shadow', 'none');
			await expect(button).toHaveCSS(
				'transform',
				reducedMotion === 'reduce' ? 'none' : 'matrix(1, 0, 0, 1, 3, 3)'
			);
			await page.mouse.move(0, 0);
			await page.mouse.up();
		});
	}
}
