import type { ReactNode } from 'react';

import { cn } from './cn';

export interface SectionHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  /** 标题层级，默认 h2。 */
  as?: 'h1' | 'h2' | 'h3';
  className?: string;
  id?: string;
}

/** 区块标题：20sp 粗体 + 可选说明与右侧动作。 */
export function SectionHeader({
  title,
  description,
  action,
  as: Heading = 'h2',
  className,
  id,
}: SectionHeaderProps) {
  return (
    <div className={cn('flex items-end justify-between gap-4', className)}>
      <div className="min-w-0">
        <Heading
          id={id}
          className={cn(
            'font-bold text-text-primary',
            Heading === 'h1' ? 'text-display' : Heading === 'h2' ? 'text-title' : 'text-subtitle',
          )}
        >
          {title}
        </Heading>
        {description ? (
          <p className="mt-1.5 text-caption leading-relaxed text-text-tertiary">{description}</p>
        ) : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  );
}
