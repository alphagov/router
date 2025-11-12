const { test } = require('./fixtures');
const { expect } = require('@playwright/test');

test.describe('GOV.UK Page Renders', () => {
  test('MOT inspection manual guidance page renders with expected content', async ({ page }) => {
    const response = await page.goto('/guidance/mot-inspection-manual-for-private-passenger-and-light-commercial-vehicles');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Inspection processes and rules for car, private bus and light commercial vehicle')).toBeVisible();
  });

  test('Highway Code section page renders with expected content', async ({ page }) => {
    const response = await page.goto('/guidance/the-highway-code/rules-for-motorcyclists-83-to-88');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('On all journeys, the rider and pillion passenger on a motorcycle')).toBeVisible();
  });

  test('Visas and immigration browse page renders with expected content', async ({ page }) => {
    const response = await page.goto('/browse/visas-immigration');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Apply to visit, work, study, settle or seek asylum in the UK')).toBeVisible();
  });

  test('Driving browse page renders with expected content', async ({ page }) => {
    const response = await page.goto('/browse/driving/vehicle-tax-mot-insurance');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Driver and vehicles account: sign in or set up')).toBeVisible();
  });

  test('Employment tribunal decision page renders with expected content', async ({ page }) => {
    const response = await page.goto('/employment-tribunal-decisions/mr-g-d-monroe-v-royal-national-lifeboat-institution-rnli-6010603-slash-2024');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Read the full decision in')).toBeVisible();
  });

  test('HMRC manual section page renders with expected content', async ({ page }) => {
    const response = await page.goto('/hmrc-internal-manuals/international-exchange-of-information/ieim402040');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('It is a unique combination of letters')).toBeVisible();
  });

  test('News story page renders with expected content', async ({ page }) => {
    const response = await page.goto('/government/news/bird-flu-avian-influenza-latest-situation-in-england');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Find out about the latest bird flu situation in England and guidance for bird keepers and the public')).toBeVisible();
  });

  test('Universal Credit guide page renders with expected content', async ({ page }) => {
    const response = await page.goto('/universal-credit');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Universal Credit is a payment to help with your living costs')).toBeVisible();
  });

  test('Universal Credit how to claim page renders with expected content', async ({ page }) => {
    const response = await page.goto('/universal-credit/how-to-claim');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('You need to create an account to make a claim')).toBeVisible();
  });

  test('Licence application page renders with expected content', async ({ page }) => {
    const response = await page.goto('/apply-for-a-licence/test-licence/gds-test/apply-1');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('Next, fill in the application form on your computer')).toBeVisible();
  });

  test('Travel advice page renders with expected content', async ({ page }) => {
    const response = await page.goto('/foreign-travel-advice/france');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('No travel can be guaranteed safe')).toBeVisible();
  });

  test('Email alert signup page renders with expected content', async ({ page }) => {
    const response = await page.goto('/foreign-travel-advice/france/email-signup');

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(300);
    await expect(page.locator('body')).toBeVisible();
    await expect(page.getByText('France travel advice')).toBeVisible();
  });
});
