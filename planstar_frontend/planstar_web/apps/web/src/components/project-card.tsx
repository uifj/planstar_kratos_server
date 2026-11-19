import Link from 'next/link';

import {
  achievementPercent,
  formatPercent,
  permilleToPercent,
  type ProjectSummary,
} from '@planstar/api-client';
import { Card, Cover, Price, Progress, Tag } from '@planstar/ui';

import { deadlineText, fundingStatusMeta } from '@/lib/format';

export interface ProjectCardProps {
  project: ProjectSummary;
  /** 首屏封面图使用 eager 加载。 */
  priority?: boolean;
}

/** 企划卡片：3:4 海报 + 状态标签 + 价格 / 进度 / 支持人数。 */
export function ProjectCard({ project, priority = false }: ProjectCardProps) {
  const status = fundingStatusMeta(project.fundingStatus);
  // 服务端 achievementRate 是千分比（860 → 86%）。
  const percent =
    project.achievementRate > 0
      ? permilleToPercent(project.achievementRate)
      : achievementPercent(project.raisedAmount, project.goalAmount);

  return (
    <Link href={`/projects/${project.projectId}`} className="group block focus-visible:outline-none">
      <Card padded={false} interactive className="flex h-full flex-col overflow-hidden">
        <div className="relative">
          <Cover
            src={project.cover}
            alt={project.title}
            fallback={project.title}
            variant="poster"
            priority={priority}
          />
          <div className="absolute left-2 top-2 flex flex-wrap gap-1">
            <Tag tone={status.tone}>{status.label}</Tag>
            {project.categoryName ? <Tag tone="neutral">{project.categoryName}</Tag> : null}
          </div>
        </div>

        <div className="flex flex-1 flex-col p-3">
          <h3 className="line-clamp-2 text-body font-medium leading-snug text-text-primary transition-colors group-hover:text-brand">
            {project.title}
          </h3>
          <p className="mt-1 truncate text-caption text-text-tertiary">
            {project.creatorName || 'Planstar 创作者'}
          </p>

          <div className="mt-auto pt-2.5">
            <div className="flex items-baseline justify-between gap-2">
              <Price cents={project.minSupportAmount} size="sm" suffix="起" />
              <span className="shrink-0 text-label text-text-tertiary">
                {deadlineText(project.remainingDays, project.deadline, project.fundingStatus)}
              </span>
            </div>
            <Progress className="mt-2" value={percent} height={4} label={`${project.title} 筹资进度`} />
            <div className="mt-1.5 flex items-center justify-between text-label text-text-tertiary">
              <span className="tabular-nums">{formatPercent(percent)} 达成</span>
              <span className="tabular-nums">{project.supporterCount} 人支持</span>
            </div>
          </div>
        </div>
      </Card>
    </Link>
  );
}
