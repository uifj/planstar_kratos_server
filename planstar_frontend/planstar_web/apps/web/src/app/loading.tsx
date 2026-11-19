import { ProjectCardSkeleton } from '@planstar/ui';

/** 全局加载骨架：保持与列表页一致的栅格，减少布局跳动。 */
export default function Loading() {
  return (
    <div className="ps-container py-12" aria-busy="true" aria-live="polite">
      <div className="h-7 w-40 animate-pulse rounded-xs bg-skeleton" />
      <div className="mt-2 h-4 w-64 animate-pulse rounded-xs bg-skeleton" />
      <div className="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 lg:gap-4">
        {Array.from({ length: 8 }).map((_, index) => (
          <ProjectCardSkeleton key={index} />
        ))}
      </div>
    </div>
  );
}
