const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('JSON API Tests', () => {
  test('bank holidays JSON endpoint returns valid JSON', async ({ page }) => {
    // Frontend app
    const response = await page.goto('/bank-holidays.json');

    // Verify 200 status code
    expect(response.status()).toBe(200);

    // Get response body as text
    const body = await response.text();

    // Verify the response is valid JSON by parsing it
    let jsonData;
    expect(() => {
      jsonData = JSON.parse(body);
    }).not.toThrow();

    // Verify Content-Type header indicates JSON
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Verify the JSON has expected structure for bank holidays
    expect(jsonData).toHaveProperty('england-and-wales');
  });

  test('search autocomplete JSON endpoint returns valid JSON', async ({ page }) => {
    // Finder-frontend app
    const response = await page.goto('/api/search/autocomplete.json?q=bernar');

    // Verify 200 status code
    expect(response.status()).toBe(200);

    // Get response body as text
    const body = await response.text();

    // Verify the response is valid JSON by parsing it
    let jsonData;
    expect(() => {
      jsonData = JSON.parse(body);
    }).not.toThrow();

    // Verify Content-Type header indicates JSON
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Verify the JSON has expected autocomplete structure
    expect(jsonData).toHaveProperty('suggestions');
    expect(Array.isArray(jsonData.suggestions)).toBe(true);
  });

  test('search API v2 JSON endpoint returns valid JSON', async ({ request }) => {
    // Search-api-v2
    // Note: This is a different domain so using request context instead of page

    // Extract environment from BASE_URL and construct appropriate search API URL
    const baseUrl = process.env.BASE_URL || 'https://www.gov.uk';
    let searchApiHost;

    if (baseUrl.includes('integration.publishing.service.gov.uk')) {
      searchApiHost = 'https://search.integration.publishing.service.gov.uk';
    } else if (baseUrl.includes('staging.publishing.service.gov.uk')) {
      searchApiHost = 'https://search.staging.publishing.service.gov.uk';
    } else {
      // Production environment (www.gov.uk)
      searchApiHost = 'https://search.publishing.service.gov.uk';
    }

    const response = await request.get(`${searchApiHost}/v0_1/search.json`);

    // Verify 200 status code
    expect(response.status()).toBe(200);

    // Get response body as text
    const body = await response.text();

    // Verify the response is valid JSON by parsing it
    let jsonData;
    expect(() => {
      jsonData = JSON.parse(body);
    }).not.toThrow();

    // Verify Content-Type header indicates JSON
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Verify the JSON has expected search result structure
    expect(jsonData).toHaveProperty('results');
  });

  test('content API endpoint returns valid JSON', async ({ page }) => {
    // Content-store app
    const response = await page.goto('/api/content');

    // Verify 200 status code
    expect(response.status()).toBe(200);

    // Get response body as text
    const body = await response.text();

    // Verify the response is valid JSON by parsing it
    let jsonData;
    expect(() => {
      jsonData = JSON.parse(body);
    }).not.toThrow();

    // Verify Content-Type header indicates JSON
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Verify the JSON has expected content structure
    expect(jsonData).toBeDefined();
  });

  test('search API endpoint returns valid JSON', async ({ page }) => {
    // Search-API app
    const response = await page.goto('/api/search.json?q=simon');

    // Verify 200 status code
    expect(response.status()).toBe(200);

    // Get response body as text
    const body = await response.text();

    // Verify the response is valid JSON by parsing it
    let jsonData;
    expect(() => {
      jsonData = JSON.parse(body);
    }).not.toThrow();

    // Verify Content-Type header indicates JSON
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Verify the JSON has expected search result structure
    expect(jsonData).toHaveProperty('results');
  });

  test('organisations API endpoint returns valid JSON', async ({ page }) => {
    // Collections app
    const response = await page.goto('/api/organisations');

    // Verify 200 status code
    expect(response.status()).toBe(200);

    // Get response body as text
    const body = await response.text();

    // Verify the response is valid JSON by parsing it
    let jsonData;
    expect(() => {
      jsonData = JSON.parse(body);
    }).not.toThrow();

    // Verify Content-Type header indicates JSON
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Verify the JSON has expected organisations structure
    expect(jsonData).toHaveProperty('results');
    expect(Array.isArray(jsonData.results)).toBe(true);
  });
});
