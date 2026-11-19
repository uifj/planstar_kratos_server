import { expect, test } from '@playwright/test';

/**
 * 官网冒烟测试。
 *
 * 后端在 E2E 环境中不可用，因此只断言页面结构、导航与降级文案，
 * 不依赖真实接口数据。
 */
test.describe('官网冒烟', () => {
  test('首页：品牌、首屏主张、导航与页脚', async ({ page }) => {
    await page.goto('/');

    await expect(page).toHaveTitle(/Planstar/);
    await expect(page.getByRole('link', { name: 'Planstar 企划星球 首页' })).toBeVisible();
    await expect(page.getByRole('heading', { level: 1 })).toContainText('让每一个好企划，被看见');
    await expect(page.getByRole('link', { name: '发现企划' }).first()).toBeVisible();
    await expect(page.getByRole('contentinfo')).toContainText('Planstar 企划星球');
    // 服务保障条在首屏即可见
    await expect(page.getByText('100%履约')).toBeVisible();
  });

  test('顶部导航可以进入企划列表', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '发现企划' }).click();

    await expect(page).toHaveURL(/\/projects$/);
    await expect(page.getByRole('heading', { level: 1, name: '发现企划' })).toBeVisible();
  });

  test('企划列表页支持分类与排序筛选', async ({ page }) => {
    await page.goto('/projects');

    await expect(page.getByText('分类', { exact: true })).toBeVisible();
    await expect(page.getByText('排序', { exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: '即将结束' })).toBeVisible();
  });

  test('榜单页展示三类榜单入口', async ({ page }) => {
    await page.goto('/rankings');

    await expect(page.getByRole('heading', { level: 1, name: '企划榜单' })).toBeVisible();
    await expect(page.getByRole('link', { name: '热度榜' })).toBeVisible();
    await expect(page.getByRole('link', { name: '达成榜' })).toBeVisible();
    await expect(page.getByRole('link', { name: '冲刺榜' })).toBeVisible();
  });

  test('关于页展示平台说明', async ({ page }) => {
    await page.goto('/about');

    await expect(page.getByRole('heading', { level: 1 })).toContainText('让每一个好企划，被看见');
    await expect(page.getByRole('heading', { name: '企划如何运转' })).toBeVisible();
    await expect(page.getByRole('main').getByRole('heading', { name: '平台保障' })).toBeVisible();
  });

  test('未知路径返回 404 页面', async ({ page }) => {
    const response = await page.goto('/this-page-does-not-exist');

    expect(response?.status()).toBe(404);
    await expect(page.getByText('页面不存在或已下线')).toBeVisible();
  });
});
