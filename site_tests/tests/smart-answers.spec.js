const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('Smart Answers Tests', () => {
  test('/check-uk-visa smart answers flow completes successfully', async ({ page }) => {
    // Navigate to the UK visa checker
    const response = await page.goto('/check-uk-visa');
    expect(response.status()).toBe(200);

    // Dismiss cookie banner if present (before any other interactions)
    try {
      const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
      await rejectCookiesButton.click({ timeout: 5000 });
      // Wait for banner to be hidden
      await page.getByRole('button', { name: 'Hide cookie message' }).click();
    } catch (e) {
      // Cookie banner not present, continue
    }

    // Verify initial page content
    await expect(page.locator('body')).toContainText('You may need a visa to come to the UK to visit, study or work.');

    // Click 'Start now' button and wait for navigation
    await page.getByRole('button', { name: 'Start now' }).click();
    await page.waitForLoadState('networkidle');

    // Click first 'Continue' button and wait for navigation
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.waitForLoadState('networkidle');

    // Select 'Study' radio button and click 'Continue'
    await page.locator('input[type=radio][name=response][value=study]').check();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.waitForLoadState('networkidle');

    // Select '6 months or less' radio button and click 'Continue'
    await page.locator('input[type=radio][name=response][value=six_months_or_less]').check();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.waitForLoadState('networkidle');

    // Verify we're on the results page
    await expect(page.locator('h1')).toContainText('Information based on your answers');

    // Verify final page contains expected text
    await expect(page.locator('h2:has-text("need a visa to come to the UK")')).toBeVisible();
  });
});
