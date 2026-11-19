import { defineConfig, devices } from '@playwright/test';

const PORT = 3100;

/**
 * E2E 配置：先 build 再起生产服务。
 *
 * 后端不可用时页面会渲染统一的错误/空状态，因此冒烟用例只断言
 * 导航、标题、页脚等结构性内容，不依赖真实数据。
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    trace: 'on-first-retry',
    locale: 'zh-CN',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: `pnpm exec next start -p ${PORT}`,
    url: `http://127.0.0.1:${PORT}`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: {
      PLANSTAR_API_BASE: 'http://127.0.0.1:65534/',
      PLANSTAR_API_TIMEOUT_MS: '2000',
    },
  },
});
