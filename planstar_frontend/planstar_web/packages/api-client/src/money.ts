/**
 * 金额工具。
 *
 * 服务端所有金额字段都是「分」；展示层统一在这里转换，
 * 禁止在组件里手写 `/ 100`。
 */

const yuanFormatter = new Intl.NumberFormat('zh-CN', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});

const compactFormatter = new Intl.NumberFormat('zh-CN', {
  notation: 'compact',
  maximumFractionDigits: 1,
});

/** 分 → 元（保留两位小数的数值）。 */
export function centsToYuan(cents: number): number {
  if (!Number.isFinite(cents)) return 0;
  return Math.round(cents) / 100;
}

/** 分 → `1,280` 形式的元字符串（默认不带符号）。 */
export function formatYuan(cents: number, options: { symbol?: boolean } = {}): string {
  const value = yuanFormatter.format(centsToYuan(cents));
  return options.symbol === false ? value : `¥${value}`;
}

/** 分 → `¥12.8万` / `¥1.2亿` 的紧凑写法，用于筹资总额等大数字。 */
export function formatCompactYuan(cents: number): string {
  return `¥${compactFormatter.format(centsToYuan(cents))}`;
}

/** 数值 → 整数百分比（0–100+），非法值返回 0。 */
export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '0%';
  return `${Math.round(value)}%`;
}

/**
 * 千分比 → 百分比。
 *
 * 服务端 `achievementRate` / `fulfillmentRate` / 达成榜 `metricValue`
 * 均以千分比下发（860 表示 86%），展示前必须除以 10。
 */
export function permilleToPercent(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return value / 10;
}

/**
 * 达成率百分比。
 *
 * 服务端已给 `achievementRate`，但部分老数据缺失，这里用
 * 已筹 / 目标兜底；目标为 0 时返回 0。
 */
export function achievementPercent(raisedAmount: number, goalAmount: number): number {
  if (goalAmount <= 0) return 0;
  return Math.min(Math.max((raisedAmount / goalAmount) * 100, 0), 9999);
}
