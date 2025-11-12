const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('404 Tests', () => {
  test('404 page returns correct status code and renders error page', async ({ page }) => {
    const response = await page.goto('/really-not-here');

    // Verify 404 status code is returned
    expect(response.status()).toBe(404);

    // Verify the error page renders correctly (body visible)
    await expect(page.locator('body')).toBeVisible();

    // Search for "Page not found" text on the page
    await expect(page.locator('body')).toContainText('Page not found');
  });
});
