const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './test',
  fullyParallel: true,
  workers: 2,
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL: 'http://127.0.0.1:4187',
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'npm run dev:static',
    url: 'http://127.0.0.1:4187/',
    reuseExistingServer: false,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
