import type { HTMLAttributes } from 'react';

import { formatCompactYuan, formatYuan } from '@planstar/api-client';
import { cn } from './cn';

export interface PriceProps extends Omit<HTMLAttributes<HTMLSpanElement>, 'children'> {
  /** 金额（分）。 */
  cents: number;
  size?: 'sm' | 'md' | 'lg';
  symbol?: boolean;
  compact?: boolean;
  suffix?: string;
}

const sizeClass = {
  sm: 'text-caption',
  md: 'text-subtitle',
  lg: 'text-display',
} as const;

/**
 * 价格：等宽数字（tabular-nums）+ 品牌价格色。
 *
 * 与 Flutter `PsPrice` 一致，避免列表中不同位数价格左右跳动。
 */
export function Price({
  cents,
  size = 'md',
  symbol = true,
  compact = false,
  suffix,
  className,
  ...rest
}: PriceProps) {
  const text = compact ? formatCompactYuan(cents) : formatYuan(cents, { symbol });
  return (
    <span
      className={cn('inline-flex items-baseline gap-1 font-bold tabular-nums text-price', sizeClass[size], className)}
      {...rest}
    >
      {text}
      {suffix ? <span className="text-caption font-normal text-text-tertiary">{suffix}</span> : null}
    </span>
  );
}
