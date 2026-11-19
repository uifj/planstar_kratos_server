import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { ProjectSummary } from '@planstar/api-client';

import { ProjectCard } from './project-card';

const project: ProjectSummary = {
  projectId: 1,
  projectNo: 'PS-0001',
  title: '星河 Live 企划',
  subtitle: '一场关于星空的现场演出',
  cover: '',
  categoryId: 2,
  categoryName: '演出',
  projectType: 'offline',
  creatorId: 9,
  creatorName: '星野工作室',
  goalAmount: 100_000,
  raisedAmount: 42_000,
  supporterCount: 128,
  deadline: Math.floor(Date.now() / 1000) + 86_400 * 5,
  remainingDays: 5,
  fundingStatus: 'ongoing',
  achievementRate: 420,
  minSupportAmount: 8_800,
  tags: ['Live', '星空'],
  isFeatured: true,
};

describe('ProjectCard', () => {
  it('渲染企划关键信息并链接到详情页', () => {
    render(<ProjectCard project={project} />);

    expect(screen.getByRole('link')).toHaveAttribute('href', '/projects/1');
    expect(screen.getByRole('heading', { name: '星河 Live 企划' })).toBeInTheDocument();
    expect(screen.getByText('筹资中')).toBeInTheDocument();
    expect(screen.getByText('¥88')).toBeInTheDocument();
    expect(screen.getByText('星野工作室')).toBeInTheDocument();
    expect(screen.getByText('42% 达成')).toBeInTheDocument();
    expect(screen.getByText('128 人支持')).toBeInTheDocument();
    expect(screen.getByText('剩余 5 天')).toBeInTheDocument();
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '42');
  });

  it('缺少封面时使用品牌海报兜底', () => {
    render(<ProjectCard project={project} />);
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
    // 海报兜底会额外渲染一份企划标题。
    expect(screen.getAllByText('星河 Live 企划').length).toBeGreaterThan(1);
    expect(screen.getByText('PLANSTAR')).toBeInTheDocument();
  });
});
