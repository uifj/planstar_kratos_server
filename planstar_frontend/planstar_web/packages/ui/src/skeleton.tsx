import { cn } from './cn';

export interface SkeletonProps {
  className?: string;
}

/** 骨架屏占位块。 */
export function Skeleton({ className }: SkeletonProps) {
  return <div aria-hidden className={cn('animate-pulse rounded-sm bg-skeleton', className)} />;
}

/** 企划卡片骨架，尺寸与真实卡片一致，避免加载时跳动。 */
export function ProjectCardSkeleton() {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-bg-card">
      <div className="aspect-[3/4] w-full animate-pulse bg-skeleton" />
      <div className="space-y-2 p-3">
        <div className="h-4 w-4/5 animate-pulse rounded-xs bg-skeleton" />
        <div className="h-3 w-2/5 animate-pulse rounded-xs bg-skeleton" />
        <div className="h-3 w-3/5 animate-pulse rounded-xs bg-skeleton" />
      </div>
    </div>
  );
}
