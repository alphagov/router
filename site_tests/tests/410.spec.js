const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('410 Gone Tests', () => {
  test('410 Gone page returns correct status code', async ({ page }) => {
    const response = await page.goto('/government/topics/transport');

    // Verify 410 status code is returned
    expect(response.status()).toBe(410);

    // Note: GOV.UK currently returns an empty response body for 410 pages
    // rather than rendering a "No longer here" page
  });
});
