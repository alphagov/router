const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('Search and Filter Tests', () => {
  test('search functionality - entering "example" shows expected results', async ({ page }) => {
    await page.goto('/search/all');

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
    }

    // Enter 'example' in the main search input field (the larger one on the page)
    const searchInput = page.locator('main input[type="search"]').first();
    await searchInput.fill('example');
    await searchInput.press('Enter');

    // Wait for the page to load the results
    await page.waitForLoadState('networkidle');

    // Verify the expected result text is visible
    await expect(page.getByText('Updated: ').first()).toBeVisible();
  });

  test('filter functionality - selecting "Money" taxon shows expected results', async ({ page }) => {
    await page.goto('/search/all');

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
    }

    // Expand the "Filter and sort" section
    await page.getByText('Filter and sort').click();

    // Expand the "Topic" section by clicking the heading
    await page.getByRole('heading', { name: 'Filter by Topic' }).click();

    // Select 'Money' from the level_one_taxon dropdown
    // Note: The page automatically applies the filter when the selection is made
    await page.selectOption('select#level_one_taxon', { label: 'Money' });

    // Wait for the page to load the filtered results
    await page.waitForLoadState('domcontentloaded');

    // Verify the expected result text is visible
    await expect(page.getByText('HMRC online services: sign in or set up an account')).toBeVisible();
  });
});
