const baseTest = require('@playwright/test');

// Generate cache-busting query string
const getCacheBustParam = () => {
  return `cache_bust=${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
};

const cacheBustEnabled = (process.env.CACHE_BUST || 'true') != 'false';

// Extend base test with cache-busting functionality
const test = baseTest.test.extend({
  page: async ({ page }, use) => {
    // Only intercept requests if CACHE_BUST isn't false
    if (!cacheBustEnabled) {
      await use(page);
      return;
    }

    // Intercept all navigation requests to add cache-busting parameter
    await page.route('**/*', async (route, request) => {
      const url = new URL(request.url());

      // Only add cache-busting if URL doesn't already have query parameters
      // and if it's a navigation request (not assets like images, css, js)
      const isNavigation = request.resourceType() === 'document';
      const hasQueryParams = url.search.length > 0;

      if (isNavigation && !hasQueryParams) {
        url.search = getCacheBustParam();
        await route.continue({ url: url.toString() });
      } else {
        await route.continue();
      }
    });

    await use(page);
  },
});

module.exports = { test };
