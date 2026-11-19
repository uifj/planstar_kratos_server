import { describe, expect, it } from 'vitest';

import {
  achievementPercent,
  centsToYuan,
  formatCompactYuan,
  formatPercent,
  formatYuan,
  permilleToPercent,
} from './money';
import { toUserMessage, ApiError } from './errors';

describe('money', () => {
  it('分转元', () => {
    expect(centsToYuan(128000)).toBe(1280);
    expect(centsToYuan(88)).toBe(0.88);
    expect(centsToYuan(Number.NaN)).toBe(0);
  });

  it('格式化元金额', () => {
    expect(formatYuan(128000)).toBe('¥1,280');
    expect(formatYuan(128000, { symbol: false })).toBe('1,280');
    expect(formatYuan(0)).toBe('¥0');
  });

  it('紧凑金额', () => {
    expect(formatCompactYuan(12_800_000)).toBe('¥12.8万');
  });

  it('达成率', () => {
    expect(achievementPercent(50_000, 100_000)).toBe(50);
    expect(achievementPercent(200_000, 0)).toBe(0);
    expect(formatPercent(66.6)).toBe('67%');
  });

  it('服务端千分比转百分比', () => {
    expect(permilleToPercent(860)).toBe(86);
    expect(permilleToPercent(1830)).toBe(183);
    expect(permilleToPercent(Number.NaN)).toBe(0);
  });
});

describe('toUserMessage', () => {
  it('网络错误给出可读文案', () => {
    const error = new ApiError({ status: 0, code: 'NETWORK_ERROR', message: '', kind: 'network' });
    expect(toUserMessage(error)).toBe('无法连接到服务，请稍后重试');
  });

  it('普通 Error 直接透出 message', () => {
    expect(toUserMessage(new Error('boom'))).toBe('boom');
  });

  it('未知值兜底', () => {
    expect(toUserMessage(null)).toContain('未知错误');
  });
});
