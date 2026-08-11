import { expect, test } from '@playwright/test';

test('index page has expected h1', async ({ page }) => {
	await page.goto('/');
	await expect(
		page.getByRole('heading', {
			name: "Welcome to the UMass Amherst Makerspace's Online Portal"
		})
	).toBeVisible();
});

test('hamburger toggles the drawer without breaking alternate close controls', async ({ page }) => {
	await page.goto('/');

	const toggle = page.locator('#sidebar-toggle');
	const drawer = page.locator('#sidebar2');

	await expect(drawer).toBeHidden();
	await expect(toggle).toHaveAttribute('aria-expanded', 'false');

	await toggle.click();
	await expect(drawer).toBeVisible();
	await expect(toggle).toHaveAttribute('aria-expanded', 'true');

	await toggle.click();
	await expect(drawer).toBeHidden();
	await expect(toggle).toHaveAttribute('aria-expanded', 'false');

	await toggle.click();
	await page.getByRole('button', { name: 'Close', exact: true }).click();
	await expect(drawer).toBeHidden();

	await toggle.click();
	await page.locator('div[role="presentation"]').click();
	await expect(drawer).toBeHidden();
});
