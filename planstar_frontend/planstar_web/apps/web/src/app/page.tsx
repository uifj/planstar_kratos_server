import Link from 'next/link';

import type { Metadata } from 'next';

import { buttonClass, Card, EmptyState, SectionHeader, Tag } from '@planstar/ui';

import { DataNotice } from '@/components/data-notice';
import { Hero } from '@/components/hero';
import { ProjectGrid } from '@/components/project-grid';
import { RankingBoard } from '@/components/ranking-board';
import { createServerApiClient, load } from '@/lib/api';
import { site } from '@/lib/site';

export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: `${site.name} · ${site.slogan}`,
  description: site.description,
};

export default async function HomePage() {
  const api = await createServerApiClient();
  const [banners, hot, categories, ranking, achieved] = await Promise.all([
    load(api.getBanners()),
    load(api.getHotProjects({ page: 1, pageSize: 8 })),
    load(api.getCategories()),
    load(api.getRankings({ type: 'hot', limit: 8 })),
    load(api.getProjects({ fundingStatus: 'achieved', sort: 'achievement', pageSize: 4 })),
  ]);

  return (
    <>
      <Hero banners={banners.data ?? []} />

      <section className="ps-section" aria-labelledby="hot-projects">
        <div className="ps-container">
          <SectionHeader
            id="hot-projects"
            title="热门企划"
            description="正在被支持者关注的企划，按平台热度实时排序。"
            action={
              <Link href="/projects" className={buttonClass({ variant: 'ghost', size: 'sm' })}>
                查看全部 →
              </Link>
            }
          />

          <div className="mt-6">
            {hot.error ? (
              <DataNotice message={hot.message ?? '热门企划加载失败'} />
            ) : hot.data && hot.data.length > 0 ? (
              <ProjectGrid projects={hot.data} eagerCount={4} />
            ) : (
              <EmptyState
                title="暂无热门企划"
                description="新的企划正在筹备中，稍后再来看看。"
                action={
                  <Link href="/projects" className={buttonClass({ size: 'sm', variant: 'secondary' })}>
                    浏览全部企划
                  </Link>
                }
              />
            )}
          </div>
        </div>
      </section>

      <section className="border-y border-divider bg-bg-card">
        <div className="ps-container grid gap-8 py-10 lg:grid-cols-[1.4fr_1fr] lg:py-14">
          <div className="min-w-0">
            <SectionHeader
              title="企划榜单"
              description="热度代表当下关注度，达成率与冲刺时间帮助你判断节奏。"
              action={
                <Link href="/rankings" className={buttonClass({ variant: 'ghost', size: 'sm' })}>
                  完整榜单 →
                </Link>
              }
            />
            <div className="mt-4 flex gap-2 text-caption">
              <Tag tone="brand" size="md">热度榜</Tag>
              <Link href="/rankings?type=achievement" className="rounded-xs px-2 py-1 text-text-tertiary hover:text-brand">
                达成榜
              </Link>
              <Link href="/rankings?type=ending" className="rounded-xs px-2 py-1 text-text-tertiary hover:text-brand">
                冲刺榜
              </Link>
            </div>
            <Card className="mt-4" padded={false}>
              {ranking.error ? (
                <DataNotice className="border-0" message={ranking.message ?? '榜单加载失败'} />
              ) : ranking.data && ranking.data.items.length > 0 ? (
                <div className="px-4">
                  <RankingBoard items={ranking.data.items} type="hot" />
                </div>
              ) : (
                <EmptyState className="border-0" title="榜单生成中" description="平台正在汇总最新企划数据。" />
              )}
            </Card>
          </div>

          <div>
            <SectionHeader
              title="分类"
              description="按内容类型筛选，快速找到你感兴趣的企划。"
              action={
                <Link href="/projects" className={buttonClass({ variant: 'ghost', size: 'sm' })}>
                  全部 →
                </Link>
              }
            />
            <div className="mt-4 grid grid-cols-2 gap-3">
              {categories.error ? (
                <div className="col-span-2">
                  <DataNotice message={categories.message ?? '分类加载失败'} />
                </div>
              ) : categories.data && categories.data.length > 0 ? (
                categories.data.slice(0, 8).map((category) => (
                  <Link
                    key={category.id}
                    href={`/projects?categoryId=${category.id}`}
                    className="group flex flex-col justify-between rounded-sm border border-border bg-bg-card px-4 py-3 transition-colors hover:border-brand/50"
                  >
                    <span className="text-body font-medium text-text-primary group-hover:text-brand">
                      {category.name}
                    </span>
                    <span className="mt-1 text-label uppercase tracking-wider text-text-disabled">
                      {category.enName || `${category.subCategoryTags.length} 个标签`}
                    </span>
                  </Link>
                ))
              ) : (
                <div className="col-span-2">
                  <EmptyState title="暂无分类" description="运营配置后这里会展示企划分类。" />
                </div>
              )}
            </div>
          </div>
        </div>
      </section>

      <section className="ps-section" aria-labelledby="achieved-projects">
        <div className="ps-container">
          <SectionHeader
            id="achieved-projects"
            title="已达成企划"
            description="已完成筹资目标、进入交付阶段的企划。"
            action={
              <Link
                href="/projects?fundingStatus=achieved&sort=achievement"
                className={buttonClass({ variant: 'ghost', size: 'sm' })}
              >
                查看更多 →
              </Link>
            }
          />
          <div className="mt-6">
            {achieved.error ? (
              <DataNotice message={achieved.message ?? '已达成企划加载失败'} />
            ) : achieved.data && achieved.data.length > 0 ? (
              <ProjectGrid projects={achieved.data} />
            ) : (
              <EmptyState title="暂无已达成企划" description="达成目标的企划会在这里展示。" />
            )}
          </div>
        </div>
      </section>

      <section className="ps-container pb-16">
        <div className="overflow-hidden rounded-lg border border-border bg-brand px-6 py-10 text-on-brand md:px-12 md:py-12">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-label uppercase tracking-[0.24em] opacity-70">PLANSTAR</p>
              <h2 className="mt-2 text-title font-bold text-on-brand md:text-display">好企划值得被认真对待</h2>
              <p className="mt-2 max-w-xl text-caption leading-relaxed opacity-80 md:text-body">
                了解 Planstar 如何用公开进度、回报档位与履约跟踪，连接创作者与支持者。
              </p>
            </div>
            <Link
              href="/about"
              className="inline-flex h-11 shrink-0 items-center justify-center rounded-xl bg-on-brand px-6 text-body font-medium text-brand transition-opacity hover:opacity-90"
            >
              了解 Planstar
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
