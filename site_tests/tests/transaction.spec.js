const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('Transaction Pages', () => {
  test('/sign-in-universal-credit page renders with Sign in button', async ({ page }) => {
    const response = await page.goto('/sign-in-universal-credit');

    // Verify page renders successfully
    expect(response.status()).toBe(200);

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
      // Wait for banner to be hidden
      await page.getByRole('button', { name: 'Hide cookie message' }).click();
    }

    // Verify the "Sign in" button is visible and clickable
    const button = page.locator('a', { hasText: 'Sign in' }).first();
    await button.scrollIntoViewIfNeeded();
    await expect(button).toBeVisible();
  });

  test('/check-mot-history page renders with Start now button', async ({ page }) => {
    const response = await page.goto('/check-mot-history');

    // Verify page renders successfully
    expect(response.status()).toBe(200);

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
      // Wait for banner to be hidden
      await page.getByRole('button', { name: 'Hide cookie message' }).click();
    }

    // Verify the "Start now" button is visible and clickable
    const button = page.locator('a', { hasText: 'Start now' }).first();
    await button.scrollIntoViewIfNeeded();
    await expect(button).toBeVisible();
  });
});
