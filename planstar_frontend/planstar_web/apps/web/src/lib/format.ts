import type { FundingStatus } from '@planstar/api-client';
import type { TagTone } from '@planstar/ui';

/**
 * 后端时间字段是 Unix 秒（int64），部分聚合字段可能是毫秒；
 * 统一按数量级判断，避免出现 1970 年。
 */
export function toDate(value?: number | null): Date | null {
  if (value === undefined || value === null || value <= 0) return null;
  const ms = value < 1_000_000_000_000 ? value * 1000 : value;
  const date = new Date(ms);
  return Number.isNaN(date.getTime()) ? null : date;
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

/** 2026.11.17 */
export function formatDate(value?: number | null): string {
  const date = toDate(value);
  if (!date) return '待定';
  return `${date.getFullYear()}.${pad(date.getMonth() + 1)}.${pad(date.getDate())}`;
}

/** 2026.11.17 20:30 */
export function formatDateTime(value?: number | null): string {
  const date = toDate(value);
  if (!date) return '待定';
  return `${formatDate(value)} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** 剩余天数：优先使用服务端 remainingDays，缺省时按截止时间计算。 */
export function remainingDays(remaining?: number, deadline?: number): number | null {
  if (typeof remaining === 'number' && Number.isFinite(remaining)) return remaining;
  const date = toDate(deadline);
  if (!date) return null;
  const diff = date.getTime() - Date.now();
  return Math.max(Math.ceil(diff / 86_400_000), 0);
}

export function deadlineText(
  remaining?: number,
  deadline?: number,
  fundingStatus?: string,
): string {
  const days = remainingDays(remaining, deadline);
  if (days === null) return '时间待定';
  if (days <= 0) {
    return fundingStatus === 'ongoing' || fundingStatus === 'preheating' ? '今日截止' : '已结束';
  }
  return `剩余 ${days} 天`;
}

export interface StatusMeta {
  label: string;
  tone: TagTone;
}

const statusMeta: Record<string, StatusMeta> = {
  preheating: { label: '预热中', tone: 'warning' },
  ongoing: { label: '筹资中', tone: 'brand' },
  achieved: { label: '已达成', tone: 'success' },
  delivered: { label: '已交付', tone: 'success' },
  failed: { label: '未达成', tone: 'neutral' },
};

/** 筹资状态 → 文案与标签色；未知状态兜底为中性标签。 */
export function fundingStatusMeta(status?: FundingStatus | string): StatusMeta {
  if (!status) return { label: '状态未知', tone: 'neutral' };
  return statusMeta[status] ?? { label: status, tone: 'neutral' };
}

/** 截断长文本，避免卡片高度失控。 */
export function truncate(text: string, max = 80): string {
  return text.length > max ? `${text.slice(0, max)}…` : text;
}
