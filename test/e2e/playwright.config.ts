import { defineConfig } from '@playwright/test'

// BASE_URL is provided by `make e2e`, which starts a real panel-api serving
// the production frontend build.
const baseURL = process.env.BASE_URL ?? 'http://127.0.0.1:8117'

export default defineConfig({
  testDir: '.',
  timeout: 30_000,
  retries: 0,
  workers: 1,
  use: {
    baseURL,
    trace: 'retain-on-failure',
  },
  reporter: [['list']],
})
