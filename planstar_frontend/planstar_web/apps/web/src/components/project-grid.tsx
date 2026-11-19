import type { ProjectSummary } from '@planstar/api-client';

import { ProjectCard } from './project-card';

export interface ProjectGridProps {
  projects: ProjectSummary[];
  /** 首屏前 N 张卡片优先加载。 */
  eagerCount?: number;
  className?: string;
}

/** 企划栅格：移动端两列、平板三列、桌面四列。 */
export function ProjectGrid({ projects, eagerCount = 0, className }: ProjectGridProps) {
  return (
    <div className={className ?? 'grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 lg:gap-4'}>
      {projects.map((project, index) => (
        <ProjectCard key={project.projectId} project={project} priority={index < eagerCount} />
      ))}
    </div>
  );
}
