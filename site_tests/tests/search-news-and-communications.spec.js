const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('GOV.UK News and Communications Search Tests', () => {
  test('search functionality - entering "example" shows expected results', async ({ page }) => {
    await page.goto('/search/news-and-communications');

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
    }

    // Enter 'example' in the main search input field (the finder search box)
    const searchInput = page.locator('input[type="search"][name="keywords"]#finder-keyword-search');
    await searchInput.fill('example');
    await searchInput.press('Enter');

    // Wait for the page to load the results
    await page.waitForLoadState('networkidle');

    // Verify the expected result text is visible
    await expect(page.getByText('Illustrative example of a sample single line diagram')).toBeVisible();
  });

  test('filter functionality - selecting "Environment" taxon and "Updated (oldest)" shows expected results', async ({ page }) => {
    await page.goto('/search/news-and-communications');

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
    }

    // Wait for page to fully load
    await page.waitForLoadState('networkidle');

    // Select 'Environment' from the level_one_taxon dropdown
    // Use force option to interact with potentially hidden elements
    await page.locator('select#level_one_taxon').selectOption({ label: 'Environment' }, { force: true });

    // Wait for page to update after first filter
    await page.waitForLoadState('networkidle');

    // Select 'Updated (oldest)' from the order dropdown
    await page.locator('select#order').selectOption({ label: 'Updated (oldest)' }, { force: true });

    // Wait for page to update after second filter
    await page.waitForLoadState('networkidle');

    // Verify the expected result text is visible
    await expect(page.getByText('Welsh guide to legislation')).toBeVisible();
  });
});
