import type { ReactNode } from 'react';

import { cn } from './cn';

export interface StatProps {
  label: ReactNode;
  value: ReactNode;
  hint?: ReactNode;
  className?: string;
  align?: 'left' | 'center';
}

/** 指标块：数值 20sp 粗体，名称 12sp 次要色。 */
export function Stat({ label, value, hint, className, align = 'left' }: StatProps) {
  return (
    <div className={cn('min-w-0', align === 'center' && 'text-center', className)}>
      <div className="text-title font-bold tabular-nums text-text-primary">{value}</div>
      <div className="mt-0.5 text-caption text-text-tertiary">{label}</div>
      {hint ? <div className="mt-1 text-label text-text-disabled">{hint}</div> : null}
    </div>
  );
}
