const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('GOV.UK Redirect Tests', () => {

  test.describe('Router Redirects', () => {
    test('/brexit redirects correctly', async ({ page }) => {
      // Capture the redirect response before following it
      let redirectResponse = null;
      page.on('response', response => {
        if (response.url().endsWith('/brexit') && [301, 302, 303, 307, 308].includes(response.status())) {
          redirectResponse = response;
        }
      });

      await page.goto('/brexit');

      // Verify we got a redirect from /brexit
      expect(redirectResponse).not.toBeNull();
      expect([301, 307]).toContain(redirectResponse.status());

      // Verify the final URL (may redirect through /government/brexit to collections page)
      expect(page.url()).toMatch(/\/government\/(brexit|collections\/brexit-guidance)/);

      // Ensure the page loaded successfully after redirect
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('App Redirects (Collections)', () => {
    test('/government/brexit redirects to /government/collections/brexit-guidance with 307', async ({ page }) => {
      // Capture the redirect response before following it
      let redirectResponse = null;
      page.on('response', response => {
        if (response.url().includes('/government/brexit') &&
            !response.url().includes('collections') &&
            [301, 302, 303, 307, 308].includes(response.status())) {
          redirectResponse = response;
        }
      });

      await page.goto('/government/brexit');

      // Verify we got a redirect
      expect(redirectResponse).not.toBeNull();
      expect(redirectResponse.status()).toBe(307);

      // Verify the final URL after redirect
      expect(page.url()).toContain('/government/collections/brexit-guidance');

      // Ensure the page loaded successfully after redirect
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('External Redirects', () => {
    test('/prepare redirects to external campaign site with 301', async ({ page }) => {
      // Capture the redirect response before following it
      let redirectResponse = null;
      page.on('response', response => {
        const url = new URL(response.url());
        if (url.pathname === '/prepare' && [301, 302, 303, 307, 308].includes(response.status())) {
          redirectResponse = response;
        }
      });

      await page.goto('/prepare');

      // Verify we got a redirect
      expect(redirectResponse).not.toBeNull();
      expect(redirectResponse.status()).toBe(301);

      // Verify redirect target is the external campaign site
      expect(page.url()).toMatch(/^https:\/\/prepare\.campaign\.gov\.uk/);

      // Ensure the external page loaded successfully
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Redirect Chain Verification', () => {
    test('/brexit follows complete redirect chain correctly', async ({ page, context }) => {
      // Track all requests to verify redirect chain
      const redirectChain = [];

      page.on('response', response => {
        if ([301, 302, 303, 307, 308].includes(response.status())) {
          redirectChain.push({
            url: response.url(),
            status: response.status()
          });
        }
      });

      await page.goto('/brexit');

      // Verify we captured redirects
      expect(redirectChain.length).toBeGreaterThan(0);

      // Verify final destination
      expect(page.url()).toContain('/government/');
    });
  });
});
