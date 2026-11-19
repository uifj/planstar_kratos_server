import type { HTMLAttributes, ReactNode } from 'react';

import { cn } from './cn';

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  /** 可交互卡片：hover 时描边变为品牌色。 */
  interactive?: boolean;
  padded?: boolean;
}

/** 卡片：白底、12px 圆角、1px 细描边、无阴影（与 Flutter CardTheme 一致）。 */
export function Card({
  children,
  className,
  interactive = false,
  padded = true,
  ...rest
}: CardProps) {
  return (
    <div
      className={cn(
        'rounded-md border border-border bg-bg-card',
        padded && 'p-4',
        interactive &&
          'transition-colors duration-150 hover:border-brand/45 focus-within:border-brand/45',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  );
}
