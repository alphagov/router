const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('Highway Code guidance page', () => {
  const pageUrl = '/guidance/the-highway-code';

  test('page loads successfully and contains expected content', async ({ page }) => {
    const response = await page.goto(pageUrl);

    // Check 200 status code
    expect(response.status()).toBe(200);

    // Verify the page contains the essential reading text
    await expect(page.locator('body')).toContainText('The Highway Code is essential reading for all road users');
  });

  test('search functionality redirects correctly with query parameters', async ({ page }) => {
    await page.goto(pageUrl);

    // Find the search input field and verify its attributes
    const searchInput = page.locator('input[type="search"][title="Search"]');
    await expect(searchInput).toBeVisible();

    // Enter search term and submit
    await searchInput.fill('penalties');
    await searchInput.press('Enter');

    // Wait for navigation to complete
    await page.waitForURL(/\/search\/all/);

    // Verify the URL contains both required query parameters
    const currentUrl = page.url();
    expect(currentUrl).toContain('/search/all');
    expect(currentUrl).toContain('the-highway-code');
    expect(currentUrl).toContain('penalties');
  });
});
