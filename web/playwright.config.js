const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './test',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'python3 test/server.py',
    url: 'http://127.0.0.1:4173/',
    reuseExistingServer: false,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
