import { describe, expect, it } from 'vitest';

import { deadlineText, formatDate, formatDateTime, fundingStatusMeta, toDate, truncate } from './format';

describe('时间格式化', () => {
  it('兼容秒与毫秒时间戳', () => {
    const local = new Date(2026, 10, 17, 9, 5, 0);
    const seconds = Math.floor(local.getTime() / 1000);

    expect(formatDate(seconds)).toBe('2026.11.17');
    expect(formatDateTime(seconds)).toBe('2026.11.17 09:05');
    expect(formatDate(local.getTime())).toBe('2026.11.17');
  });

  it('空值不产生 1970', () => {
    expect(toDate(0)).toBeNull();
    expect(formatDate(undefined)).toBe('待定');
  });
});

describe('截止时间文案', () => {
  it('优先使用服务端 remainingDays', () => {
    expect(deadlineText(3, 0)).toBe('剩余 3 天');
    expect(deadlineText(0, 0)).toBe('已结束');
    expect(deadlineText(0, 0, 'ongoing')).toBe('今日截止');
  });

  it('缺省时按 deadline 计算剩余天数', () => {
    const deadline = Math.floor(Date.now() / 1000) + 86_400 * 2 + 60;
    expect(deadlineText(undefined, deadline)).toBe('剩余 3 天');
  });
});

describe('筹资状态', () => {
  it('已知状态映射文案与色调', () => {
    expect(fundingStatusMeta('ongoing')).toEqual({ label: '筹资中', tone: 'brand' });
    expect(fundingStatusMeta('achieved')).toEqual({ label: '已达成', tone: 'success' });
  });

  it('未知状态兜底', () => {
    expect(fundingStatusMeta('unknown')).toEqual({ label: 'unknown', tone: 'neutral' });
    expect(fundingStatusMeta(undefined).label).toBe('状态未知');
  });
});

describe('truncate', () => {
  it('超长文本截断并追加省略号', () => {
    expect(truncate('abc', 2)).toBe('ab…');
    expect(truncate('abc', 4)).toBe('abc');
  });
});
