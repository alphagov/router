const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('GOV.UK Homepage', () => {
  test('homepage loads successfully with search autocomplete', async ({ page }) => {
    // Skip this test if not running against production
    const baseUrl = process.env.BASE_URL || 'https://www.gov.uk';
    test.skip(baseUrl !== 'https://www.gov.uk', 'Homepage test only runs against production');

    // Navigate to homepage and verify it loads with 200 status
    const response = await page.goto('/');
    expect(response.status()).toBe(200);

    // Verify the page title
    await expect(page).toHaveTitle('Welcome to GOV.UK');

    // Dismiss cookie banner if present (before any other interactions)
    try {
      const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
      await rejectCookiesButton.click({ timeout: 5000 });
      // Wait for banner to be hidden
      await page.getByRole('button', { name: 'Hide cookie message' }).click();
    } catch (e) {
      // Cookie banner not present, continue
    }

    // Find the search input box within the homepage header
    const searchInput = page.locator('.homepage-header__search input[type="search"]');
    await expect(searchInput).toBeVisible();

    // Enter 'test' into the search input
    await searchInput.fill('test');

    // Wait for autocomplete suggestions to appear and verify 'book practical driving' is suggested
    // The autocomplete typically appears in a dropdown or suggestion list
    await page.waitForTimeout(1000); // Give autocomplete time to load

    // Check that 'book practical driving' text appears on the page
    await expect(page.getByText(/book.*practical.*driving/i)).toBeVisible();
  });
});
