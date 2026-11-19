import type { ReactNode } from 'react';

import { cn } from './cn';

export interface EmptyStateProps {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  /** 图标使用 emoji 或外部 SVG，保持零依赖。 */
  icon?: ReactNode;
  className?: string;
}

/** 空状态 / 错误状态：居中、轻量、不抢主内容。 */
export function EmptyState({ title, description, action, icon, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 rounded-md border border-dashed border-border bg-bg-card px-6 py-14 text-center',
        className,
      )}
    >
      {icon ? <div className="text-3xl leading-none opacity-80">{icon}</div> : null}
      <p className="text-subtitle font-medium text-text-primary">{title}</p>
      {description ? (
        <p className="max-w-md text-caption leading-relaxed text-text-tertiary">{description}</p>
      ) : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}
