const { defineConfig } = require('@playwright/test');
module.exports = defineConfig({
  testDir: './test', testMatch: 'service-status-reload.spec.js',
  workers: 1, fullyParallel: false, retries: 0, timeout: 90000,
  use: { baseURL: `http://127.0.0.1:${process.env.CRAFTSKY_STATUS_APP_PORT || 4175}`, trace: 'retain-on-failure' },
  outputDir: 'test-results/status-reload',
  webServer: {
    command: 'python3 test/service-status-reload-server.py',
    url: `http://127.0.0.1:${process.env.CRAFTSKY_STATUS_APP_PORT || 4175}/__requests`,
    reuseExistingServer: false, stdout: 'ignore', stderr: 'pipe',
  },
});
