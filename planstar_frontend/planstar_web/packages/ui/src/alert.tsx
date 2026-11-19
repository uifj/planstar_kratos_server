import type { ReactNode } from 'react';

import { cn } from './cn';

export type AlertTone = 'info' | 'warning' | 'error' | 'success';

const toneClass: Record<AlertTone, string> = {
  info: 'border-info/30 bg-info/5 text-text-secondary',
  warning: 'border-warning/30 bg-warning-bg text-text-secondary',
  error: 'border-error/30 bg-error/5 text-text-secondary',
  success: 'border-success/30 bg-success/5 text-text-secondary',
};

export interface AlertProps {
  tone?: AlertTone;
  title?: ReactNode;
  children?: ReactNode;
  className?: string;
}

/** 提示条：用于风险说明、退款规则、加载失败等。 */
export function Alert({ tone = 'info', title, children, className }: AlertProps) {
  return (
    <div className={cn('rounded-sm border px-4 py-3 text-caption leading-relaxed', toneClass[tone], className)}>
      {title ? <p className="mb-1 text-body font-medium text-text-primary">{title}</p> : null}
      {children}
    </div>
  );
}
