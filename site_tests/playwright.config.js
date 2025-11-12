const { defineConfig, devices } = require('@playwright/test');

// Parse BASIC_AUTH environment variable (format: username:password)
const getHttpCredentials = () => {
  const basicAuth = process.env.BASIC_AUTH;
  if (!basicAuth) return undefined;

  const [username, password] = basicAuth.split(':', 2);
  if (!username || !password) {
    console.warn('BASIC_AUTH format should be username:password');
    return undefined;
  }

  return { username, password };
};

module.exports = defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['list']
  ],
  use: {
    baseURL: process.env.BASE_URL || 'https://www.gov.uk',
    httpCredentials: getHttpCredentials(),
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  timeout: 30000,
  expect: {
    timeout: 10000
  },

  projects: [
    {
      name: 'firefox',
      use: {
        ...devices['Desktop Firefox']
      },
    },
  ],
});
