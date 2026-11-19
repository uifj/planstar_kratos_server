import type { HTMLAttributes, ReactNode } from 'react';

import { cn } from './cn';

export type TagTone = 'brand' | 'neutral' | 'success' | 'warning' | 'danger' | 'outline';

const toneClass: Record<TagTone, string> = {
  brand: 'bg-brand-subtle text-brand',
  neutral: 'bg-bg-subtle text-text-secondary',
  success: 'bg-success/10 text-success',
  warning: 'bg-warning-bg text-warning',
  danger: 'bg-error/10 text-error',
  outline: 'border border-border text-text-secondary',
};

export interface TagProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: TagTone;
  size?: 'sm' | 'md';
  children: ReactNode;
}

/** 标签/徽标：4px 圆角，10–12sp。 */
export function Tag({ tone = 'neutral', size = 'sm', className, children, ...rest }: TagProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center justify-center gap-1 whitespace-nowrap rounded-xs font-medium',
        size === 'sm' ? 'h-5 px-1.5 text-label' : 'h-6 px-2 text-caption',
        toneClass[tone],
        className,
      )}
      {...rest}
    >
      {children}
    </span>
  );
}
