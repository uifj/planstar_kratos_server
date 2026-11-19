import Link from 'next/link';

import type { Metadata } from 'next';

import type { RankingType } from '@planstar/api-client';
import { Card, cn, EmptyState, SectionHeader, Tag } from '@planstar/ui';

import { DataNotice } from '@/components/data-notice';
import { RankingBoard } from '@/components/ranking-board';
import { createServerApiClient, load } from '@/lib/api';

export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: '企划榜单',
  description: 'Planstar 热度榜、达成榜与冲刺榜，按平台实时数据生成。',
};

const rankingTabs: { type: RankingType; label: string; description: string }[] = [
  { type: 'hot', label: '热度榜', description: '按企划热度值排序，反映当下的关注度。' },
  { type: 'achievement', label: '达成榜', description: '按筹资达成率排序，看看哪些企划率先完成目标。' },
  { type: 'ending', label: '冲刺榜', description: '按剩余时间排序，关注即将截止的企划。' },
];

type SearchParams = Record<string, string | string[] | undefined>;

function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

function parseType(value?: string): RankingType {
  return value === 'achievement' || value === 'ending' ? value : 'hot';
}

function buildHref(type: RankingType, categoryId?: number): string {
  const search = new URLSearchParams({ type });
  if (categoryId) search.set('categoryId', String(categoryId));
  return `/rankings?${search.toString()}`;
}

export default async function RankingsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const type = parseType(first(params.type));
  const categoryIdRaw = Number(first(params.categoryId));
  const categoryId = Number.isInteger(categoryIdRaw) && categoryIdRaw > 0 ? categoryIdRaw : undefined;

  const api = await createServerApiClient();
  const [ranking, categories] = await Promise.all([
    load(api.getRankings({ type, limit: 20, categoryId })),
    load(api.getCategories()),
  ]);

  const activeTab = rankingTabs.find((tab) => tab.type === type) ?? rankingTabs[0]!;

  return (
    <>
      <section className="border-b border-divider bg-bg-card">
        <div className="ps-container py-10 md:py-14">
          <SectionHeader
            as="h1"
            title="企划榜单"
            description="榜单由平台按实时数据生成，帮助你快速判断企划的节奏与关注度。"
          />

          <div className="mt-6 flex flex-wrap gap-2">
            {rankingTabs.map((tab) => (
              <Link
                key={tab.type}
                href={buildHref(tab.type, categoryId)}
                className={cn(
                  'rounded-sm px-4 py-2 text-caption transition-colors',
                  tab.type === type
                    ? 'bg-brand font-medium text-on-brand'
                    : 'bg-bg-subtle text-text-secondary hover:text-brand',
                )}
              >
                {tab.label}
              </Link>
            ))}
          </div>

          <div className="mt-3 flex flex-wrap items-center gap-2">
            <span className="text-caption text-text-tertiary">分类</span>
            <Link
              href={buildHref(type)}
              className={cn(
                'rounded-xs px-2 py-1 text-label transition-colors',
                !categoryId ? 'bg-brand-subtle text-brand' : 'text-text-tertiary hover:text-brand',
              )}
            >
              全部
            </Link>
            {(categories.data ?? []).map((category) => (
              <Link
                key={category.id}
                href={buildHref(type, category.id)}
                className={cn(
                  'rounded-xs px-2 py-1 text-label transition-colors',
                  categoryId === category.id ? 'bg-brand-subtle text-brand' : 'text-text-tertiary hover:text-brand',
                )}
              >
                {category.name}
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="ps-section">
        <div className="ps-container">
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <h2 className="text-subtitle font-bold text-text-primary">{activeTab.label}</h2>
              {ranking.data ? <Tag tone="neutral">共 {ranking.data.items.length} 个</Tag> : null}
            </div>
            <p className="hidden text-caption text-text-tertiary sm:block">{activeTab.description}</p>
          </div>

          <Card padded={false}>
            {ranking.error ? (
              <DataNotice className="border-0" message={ranking.message ?? '榜单加载失败'} />
            ) : ranking.data && ranking.data.items.length > 0 ? (
              <div className="px-4">
                <RankingBoard items={ranking.data.items} type={type} />
              </div>
            ) : (
              <EmptyState
                className="border-0"
                title="榜单暂无数据"
                description="该分类下还没有进入榜单的企划，换个分类看看。"
                action={
                  <Link href="/projects" className="text-caption text-brand hover:underline">
                    浏览全部企划 →
                  </Link>
                }
              />
            )}
          </Card>
        </div>
      </section>
    </>
  );
}
