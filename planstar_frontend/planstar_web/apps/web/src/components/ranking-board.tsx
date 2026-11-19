import Link from 'next/link';

import {
  formatPercent,
  permilleToPercent,
  type RankingItem,
  type RankingType,
} from '@planstar/api-client';
import { cn, Cover, Progress } from '@planstar/ui';

import { deadlineText } from '@/lib/format';

const metricLabel: Record<RankingType, string> = {
  hot: '热度',
  achievement: '达成率',
  ending: '剩余',
};

function formatMetric(type: RankingType, item: RankingItem): string {
  // 达成榜 metricValue 与 achievementRate 一样是千分比。
  if (type === 'achievement') return formatPercent(permilleToPercent(item.metricValue));
  if (type === 'ending') {
    return `${item.project.remainingDays > 0 ? item.project.remainingDays : 0} 天`;
  }
  return String(item.metricValue);
}

export interface RankingBoardProps {
  items: RankingItem[];
  type: RankingType;
  className?: string;
}

/** 榜单列表：前 3 名用品牌色强调，其余收敛为次要色。 */
export function RankingBoard({ items, type, className }: RankingBoardProps) {
  return (
    <ol className={cn('divide-y divide-divider', className)}>
      {items.map((item) => {
        const top = item.rank <= 3;
        return (
          <li key={item.project.projectId}>
            <Link
              href={`/projects/${item.project.projectId}`}
              className="group flex items-center gap-3 py-3 transition-colors hover:bg-bg-subtle/60 sm:gap-4"
            >
              <span
                className={cn(
                  'w-7 shrink-0 text-center text-subtitle font-bold tabular-nums',
                  top ? 'text-brand' : 'text-text-disabled',
                )}
              >
                {item.rank}
              </span>
              <div className="w-12 shrink-0 sm:w-14">
                <Cover src={item.project.cover} alt={item.project.title} ratio="3/4" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="line-clamp-1 text-body font-medium text-text-primary transition-colors group-hover:text-brand">
                  {item.project.title}
                </p>
                <p className="mt-0.5 truncate text-caption text-text-tertiary">
                  {item.project.creatorName || 'Planstar 创作者'}
                  {item.project.categoryName ? ` · ${item.project.categoryName}` : ''}
                </p>
                <Progress
                  className="mt-2 max-w-[220px]"
                  value={permilleToPercent(item.project.achievementRate)}
                  height={4}
                  label={`${item.project.title} 筹资进度`}
                />
              </div>
              <div className="shrink-0 text-right">
                <p className={cn('text-body font-bold tabular-nums', top ? 'text-price' : 'text-text-primary')}>
                  {formatMetric(type, item)}
                </p>
                <p className="mt-0.5 text-label text-text-tertiary">
                  {metricLabel[type]} ·{' '}
                  {deadlineText(
                    item.project.remainingDays,
                    item.project.deadline,
                    item.project.fundingStatus,
                  )}
                </p>
              </div>
            </Link>
          </li>
        );
      })}
    </ol>
  );
}
