const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('GOV.UK Postcode Lookup Tests', () => {
  test('/find-local-council - postcode lookup redirects to correct local council page', async ({ page }) => {
    await page.goto('/find-local-council');

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
    }

    // Verify the page loads with expected text
    await expect(page.getByText('Find the website for your local council')).toBeVisible();

    // Enter postcode SW1A 2AA (Westminster) in the postcode input
    const postcodeInput = page.locator('input#postcode');
    await postcodeInput.fill('SW1A 2AA');

    // Click the Find button to submit the form
    const findButton = page.getByRole('button', { name: 'Find' });
    await findButton.click();

    // Wait for navigation to the local council page
    await page.waitForURL('**/find-local-council/westminster');

    // Verify we're on the Westminster council page
    expect(page.url()).toContain('/find-local-council/westminster');

    // Verify the page contains the council name
    await expect(page.getByText('City of Westminster').first()).toBeVisible();
  });

  test('/register-offices - postcode lookup displays correct register office information', async ({ page }) => {
    await page.goto('/register-offices');

    // Dismiss cookie banner if present
    const rejectCookiesButton = page.getByRole('button', { name: 'Reject additional cookies' });
    if (await rejectCookiesButton.isVisible()) {
      await rejectCookiesButton.click();
    }

    // Verify the page loads with expected text
    await expect(page.getByText('You can use a register office to')).toBeVisible();

    // Enter postcode SW1A 2AA (Westminster) in the postcode input
    const postcodeInput = page.locator('input#postcode');
    await postcodeInput.fill('SW1A 2AA');

    // Click the "Find a register office" button to submit the form
    const findButton = page.getByRole('button', { name: 'Find a register office' });
    await findButton.click();

    // Wait for network activity to complete (results are loaded dynamically)
    await page.waitForLoadState('networkidle');

    // Verify the page contains the Westminster Register Office information
    await expect(page.getByText('Westminster Register Office')).toBeVisible();
  });
});
