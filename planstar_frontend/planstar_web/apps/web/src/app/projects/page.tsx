import Link from 'next/link';

import type { Metadata } from 'next';

import type { FundingStatus, ProjectSort } from '@planstar/api-client';
import { buttonClass, cn, EmptyState, SectionHeader } from '@planstar/ui';

import { DataNotice } from '@/components/data-notice';
import { ProjectGrid } from '@/components/project-grid';
import { createServerApiClient, load } from '@/lib/api';

export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: '发现企划',
  description: '按分类、筹资状态与排序浏览 Planstar 上的企划，找到值得支持的创作。',
};

const PAGE_SIZE = 12;

const statusOptions: { value: string; label: string }[] = [
  { value: '', label: '全部状态' },
  { value: 'ongoing', label: '筹资中' },
  { value: 'preheating', label: '预热中' },
  { value: 'achieved', label: '已达成' },
  { value: 'delivered', label: '已交付' },
];

const sortOptions: { value: string; label: string }[] = [
  { value: '', label: '推荐排序' },
  { value: 'new', label: '最新发布' },
  { value: 'ending', label: '即将结束' },
  { value: 'achievement', label: '达成率' },
];

type SearchParams = Record<string, string | string[] | undefined>;

function first(value: string | string[] | undefined): string | undefined {
  if (Array.isArray(value)) return value[0];
  return value;
}

function toPositiveInt(value: string | undefined, fallback: number): number {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
}

function buildHref(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '') continue;
    search.set(key, String(value));
  }
  const qs = search.toString();
  return qs ? `/projects?${qs}` : '/projects';
}

export default async function ProjectsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const categoryId = first(params.categoryId) ? toPositiveInt(first(params.categoryId), 0) : undefined;
  const fundingStatus = first(params.fundingStatus) || undefined;
  const sort = first(params.sort) || undefined;
  const page = toPositiveInt(first(params.page), 1);

  const api = await createServerApiClient();
  const [categories, projects] = await Promise.all([
    load(api.getCategories()),
    load(
      api.getProjects({
        page,
        pageSize: PAGE_SIZE,
        categoryId: categoryId || undefined,
        fundingStatus: fundingStatus as FundingStatus | undefined,
        sort: sort as ProjectSort | undefined,
      }),
    ),
  ]);

  const hasNext = (projects.data?.length ?? 0) === PAGE_SIZE;
  const activeCategory = categories.data?.find((item) => item.id === categoryId);

  return (
    <>
      <section className="border-b border-divider bg-bg-card">
        <div className="ps-container py-10 md:py-14">
          <SectionHeader
            as="h1"
            title="发现企划"
            description="按分类与筹资状态筛选，支持你真正关心的创作。"
          />

          <div className="mt-6 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="w-14 shrink-0 text-caption text-text-tertiary">分类</span>
              <Link
                href={buildHref({ fundingStatus, sort })}
                className={cn(
                  'rounded-sm px-3 py-1.5 text-caption transition-colors',
                  !categoryId ? 'bg-brand text-on-brand' : 'bg-bg-subtle text-text-secondary hover:text-brand',
                )}
              >
                全部
              </Link>
              {(categories.data ?? []).map((category) => (
                <Link
                  key={category.id}
                  href={buildHref({ categoryId: category.id, fundingStatus, sort })}
                  className={cn(
                    'rounded-sm px-3 py-1.5 text-caption transition-colors',
                    categoryId === category.id
                      ? 'bg-brand text-on-brand'
                      : 'bg-bg-subtle text-text-secondary hover:text-brand',
                  )}
                >
                  {category.name}
                </Link>
              ))}
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <span className="w-14 shrink-0 text-caption text-text-tertiary">状态</span>
              {statusOptions.map((option) => (
                <Link
                  key={option.value || 'all'}
                  href={buildHref({ categoryId, fundingStatus: option.value, sort })}
                  className={cn(
                    'rounded-sm px-3 py-1.5 text-caption transition-colors',
                    (fundingStatus ?? '') === option.value
                      ? 'bg-brand text-on-brand'
                      : 'bg-bg-subtle text-text-secondary hover:text-brand',
                  )}
                >
                  {option.label}
                </Link>
              ))}
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <span className="w-14 shrink-0 text-caption text-text-tertiary">排序</span>
              {sortOptions.map((option) => (
                <Link
                  key={option.value || 'default'}
                  href={buildHref({ categoryId, fundingStatus, sort: option.value })}
                  className={cn(
                    'rounded-sm px-3 py-1.5 text-caption transition-colors',
                    (sort ?? '') === option.value
                      ? 'bg-brand-subtle font-medium text-brand'
                      : 'text-text-tertiary hover:text-brand',
                  )}
                >
                  {option.label}
                </Link>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="ps-section">
        <div className="ps-container">
          <div className="mb-5 flex items-end justify-between">
            <p className="text-caption text-text-tertiary">
              {activeCategory ? `「${activeCategory.name}」分类` : '全部企划'}
              {projects.data ? ` · 本页 ${projects.data.length} 个` : ''}
            </p>
            {page > 1 ? (
              <span className="text-label text-text-disabled">第 {page} 页</span>
            ) : null}
          </div>

          {projects.error ? (
            <DataNotice message={projects.message ?? '企划列表加载失败'} />
          ) : projects.data && projects.data.length > 0 ? (
            <>
              <ProjectGrid projects={projects.data} eagerCount={4} />
              <nav className="mt-8 flex items-center justify-between" aria-label="分页">
                {page > 1 ? (
                  <Link
                    href={buildHref({ categoryId, fundingStatus, sort, page: page - 1 })}
                    className={buttonClass({ variant: 'secondary', size: 'sm' })}
                  >
                    ← 上一页
                  </Link>
                ) : (
                  <span />
                )}
                {hasNext ? (
                  <Link
                    href={buildHref({ categoryId, fundingStatus, sort, page: page + 1 })}
                    className={buttonClass({ variant: 'secondary', size: 'sm' })}
                  >
                    下一页 →
                  </Link>
                ) : null}
              </nav>
            </>
          ) : (
            <EmptyState
              title="没有找到符合条件的企划"
              description="试试切换分类或状态，或稍后再来看看。"
              action={
                <Link href="/projects" className={buttonClass({ size: 'sm', variant: 'secondary' })}>
                  重置筛选
                </Link>
              }
            />
          )}
        </div>
      </section>
    </>
  );
}
