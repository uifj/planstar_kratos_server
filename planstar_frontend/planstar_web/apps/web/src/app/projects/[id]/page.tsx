import Link from 'next/link';

import type { Metadata } from 'next';
import { notFound } from 'next/navigation';

import {
  achievementPercent,
  formatCompactYuan,
  formatPercent,
  formatYuan,
  isNotFound,
  permilleToPercent,
  toUserMessage,
  type ProjectDetail,
} from '@planstar/api-client';
import {
  Alert,
  buttonClass,
  Card,
  Cover,
  EmptyState,
  Price,
  Progress,
  SectionHeader,
  Stat,
  Tag,
} from '@planstar/ui';

import { GuaranteeBar } from '@/components/guarantee-bar';
import { createServerApiClient } from '@/lib/api';
import { deadlineText, formatDate, formatDateTime, fundingStatusMeta } from '@/lib/format';

export const dynamic = 'force-dynamic';

interface PageProps {
  params: Promise<{ id: string }>;
}

interface DetailResult {
  project: ProjectDetail | null;
  error: unknown;
}

async function loadDetail(id: string): Promise<DetailResult> {
  const api = await createServerApiClient();
  try {
    return { project: await api.getProject(id), error: null };
  } catch (error) {
    return { project: null, error };
  }
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { id } = await params;
  const { project } = await loadDetail(id);
  if (!project) {
    return { title: '企划详情' };
  }
  const description = (project.subtitle || project.story || '').replace(/\s+/g, ' ').slice(0, 120);
  return {
    title: project.title,
    description: description || `${project.title} · Planstar 企划星球`,
    openGraph: {
      title: project.title,
      description,
      images: project.cover ? [{ url: project.cover }] : undefined,
    },
  };
}

function percentOf(project: ProjectDetail): number {
  // 服务端 achievementRate 为千分比。
  return project.achievementRate > 0
    ? permilleToPercent(project.achievementRate)
    : achievementPercent(project.raisedAmount, project.goalAmount);
}

function SupportCard({ project }: { project: ProjectDetail }) {
  const percent = percentOf(project);
  const status = fundingStatusMeta(project.fundingStatus);

  return (
    <div className="space-y-4">
      <Card className="space-y-4">
        <div className="flex items-center justify-between gap-2">
          <Tag tone={status.tone} size="md">{status.label}</Tag>
          <span className="text-caption text-text-tertiary">
            {deadlineText(project.remainingDays, project.deadline, project.fundingStatus)}
          </span>
        </div>

        <div>
          <div className="flex items-baseline justify-between">
            <p className="text-caption text-text-tertiary">已筹金额</p>
            <p className="text-caption text-text-tertiary">目标 {formatCompactYuan(project.goalAmount)}</p>
          </div>
          <p className="mt-1 text-[28px] font-bold leading-none tabular-nums text-text-primary">
            {formatYuan(project.raisedAmount)}
          </p>
        </div>

        <div>
          <Progress value={percent} height={8} label={`${project.title} 筹资进度`} />
          <div className="mt-2 flex items-center justify-between text-caption">
            <span className="font-medium tabular-nums text-brand">{formatPercent(percent)} 达成</span>
            <span className="tabular-nums text-text-tertiary">{project.supporterCount} 人支持</span>
          </div>
        </div>

        <dl className="grid grid-cols-3 gap-2 border-t border-divider pt-4">
          <div>
            <dt className="text-label text-text-tertiary">最低支持</dt>
            <dd className="mt-1">
              <Price cents={project.minSupportAmount} size="sm" />
            </dd>
          </div>
          <div>
            <dt className="text-label text-text-tertiary">回报档位</dt>
            <dd className="mt-1 text-caption font-medium tabular-nums text-text-primary">
              {project.rewardTiers.length} 个
            </dd>
          </div>
          <div>
            <dt className="text-label text-text-tertiary">截止时间</dt>
            <dd className="mt-1 text-caption font-medium tabular-nums text-text-primary">
              {formatDate(project.deadline)}
            </dd>
          </div>
        </dl>

        <p className="rounded-sm bg-bg-subtle px-3 py-2.5 text-caption leading-relaxed text-text-tertiary">
          支持与下单请在 Planstar App 内完成，官网仅展示企划进度与公开信息。
        </p>

        <Link href="/about" className={buttonClass({ block: true })}>
          了解平台保障
        </Link>
      </Card>

      <GuaranteeBar />
    </div>
  );
}

export default async function ProjectDetailPage({ params }: PageProps) {
  const { id } = await params;
  const { project, error } = await loadDetail(id);

  if (!project) {
    if (isNotFound(error)) {
      notFound();
    }

    return (
      <div className="ps-container py-16">
        <EmptyState
          title="企划详情暂时无法加载"
          description={toUserMessage(error)}
          action={
            <Link href="/projects" className={buttonClass({ size: 'sm', variant: 'secondary' })}>
              返回企划列表
            </Link>
          }
        />
      </div>
    );
  }

  const status = fundingStatusMeta(project.fundingStatus);
  const sortedFaq = [...project.faq].sort((a, b) => a.sort - b.sort);

  return (
    <div className="ps-container grid gap-8 py-8 lg:grid-cols-[minmax(0,1fr)_368px] lg:py-10">
      <article className="min-w-0">
        <nav aria-label="面包屑" className="text-caption text-text-tertiary">
          <Link href="/" className="hover:text-brand">首页</Link>
          <span className="mx-1.5">/</span>
          <Link href="/projects" className="hover:text-brand">发现企划</Link>
          <span className="mx-1.5">/</span>
          <span className="text-text-secondary">{project.title}</span>
        </nav>

        <div className="mt-4 flex flex-wrap items-center gap-2">
          <Tag tone={status.tone}>{status.label}</Tag>
          <Tag tone="neutral">{project.categoryName || '企划'}</Tag>
          {project.tags.slice(0, 3).map((tag) => (
            <Tag key={tag} tone="outline">{tag}</Tag>
          ))}
        </div>

        <h1 className="mt-3 text-title font-bold leading-snug text-text-primary md:text-[28px]">
          {project.title}
        </h1>
        {project.subtitle ? (
          <p className="mt-2 text-body leading-relaxed text-text-secondary">{project.subtitle}</p>
        ) : null}

        <div className="mt-4 flex items-center gap-2.5 text-caption text-text-tertiary">
          <span className="flex size-7 items-center justify-center rounded-full bg-brand-subtle text-label font-bold text-brand">
            {(project.creatorName || '星').slice(0, 1)}
          </span>
          <span>发起人 · {project.creatorName || 'Planstar 创作者'}</span>
          <span aria-hidden>·</span>
          <span>截止 {formatDate(project.deadline)}</span>
        </div>

        <div className="mt-5 lg:hidden">
          <SupportCard project={project} />
        </div>

        <div className="mt-5 overflow-hidden rounded-md">
          <Cover
            src={project.cover}
            alt={project.title}
            fallback={project.title}
            variant="poster"
            ratio="16/9"
            priority
          />
        </div>

        <section className="mt-8" aria-labelledby="story">
          <SectionHeader id="story" title="企划故事" />
          {project.story ? (
            <p className="mt-4 whitespace-pre-line text-body leading-[1.9] text-text-secondary">
              {project.story}
            </p>
          ) : (
            <p className="mt-4 text-caption text-text-tertiary">创作者还没有补充企划故事。</p>
          )}
        </section>

        {project.gallery.length > 0 ? (
          <section className="mt-8" aria-labelledby="gallery">
            <SectionHeader id="gallery" title="图片与现场" />
            <div className="mt-4 grid grid-cols-2 gap-3">
              {project.gallery.map((image) => (
                <Cover key={image} src={image} alt={`${project.title} 图片`} ratio="4/3" />
              ))}
            </div>
          </section>
        ) : null}

        <section className="mt-10" aria-labelledby="tiers">
          <SectionHeader
            id="tiers"
            title="回报档位"
            description="不同档位对应不同回报内容，具体以企划说明为准。"
          />
          <div className="mt-4 space-y-3">
            {project.rewardTiers.length > 0 ? (
              project.rewardTiers.map((tier) => {
                // 不限量档位（limitNum = 0）服务端也会下发 remainingNum = 0，不能据此判售罄。
                const limited = tier.limitNum > 0;
                const soldOut = limited && (tier.remainingNum ?? 0) <= 0;
                return (
                  <Card key={tier.rewardTierId} className="space-y-2">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <h3 className="text-body font-medium text-text-primary">{tier.title}</h3>
                          {tier.isEarlyBird ? <Tag tone="warning">早鸟</Tag> : null}
                          {tier.isLimited ? <Tag tone="brand">限量</Tag> : null}
                          {soldOut ? <Tag tone="neutral">已售罄</Tag> : null}
                        </div>
                        {tier.rewardType ? (
                          <p className="mt-1 text-label text-text-tertiary">回报类型 · {tier.rewardType}</p>
                        ) : null}
                      </div>
                      <Price cents={tier.price} size="md" className="shrink-0" />
                    </div>

                    {tier.items.length > 0 ? (
                      <ul className="space-y-1 text-caption text-text-secondary">
                        {tier.items.map((item) => (
                          <li key={item} className="flex gap-1.5">
                            <span aria-hidden className="text-brand">·</span>
                            <span>{item}</span>
                          </li>
                        ))}
                      </ul>
                    ) : null}

                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-divider pt-2 text-label text-text-tertiary">
                      {tier.estimatedDeliveryTime ? <span>预计交付 · {tier.estimatedDeliveryTime}</span> : null}
                      {limited && tier.remainingNum !== undefined ? (
                        <span className="tabular-nums">剩余 {Math.max(tier.remainingNum, 0)} 份</span>
                      ) : null}
                      {limited ? <span className="tabular-nums">限量 {tier.limitNum} 份</span> : null}
                    </div>
                  </Card>
                );
              })
            ) : (
              <EmptyState title="暂未公布回报档位" description="创作者正在完善回报方案，请稍后查看。" />
            )}
          </div>
        </section>

        {project.stretchGoals.length > 0 ? (
          <section className="mt-10" aria-labelledby="stretch-goals">
            <SectionHeader id="stretch-goals" title="目标与进展" description="达成基础目标后的延伸计划。" />
            <div className="mt-4 space-y-3">
              {project.stretchGoals.map((goal) => {
                const goalPercent = achievementPercent(goal.currentAmount, goal.targetAmount);
                return (
                  <Card key={goal.goalId} className="space-y-2">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-body font-medium text-text-primary">{goal.title}</p>
                      <Tag tone={goal.isAchieved ? 'success' : 'neutral'}>
                        {goal.isAchieved ? '已达成' : formatPercent(goalPercent)}
                      </Tag>
                    </div>
                    {goal.description ? (
                      <p className="text-caption text-text-tertiary">{goal.description}</p>
                    ) : null}
                    <Progress value={goalPercent} height={4} label={`${goal.title} 进度`} />
                    <p className="text-label tabular-nums text-text-tertiary">
                      {formatYuan(goal.currentAmount)} / {formatYuan(goal.targetAmount)}
                    </p>
                  </Card>
                );
              })}
            </div>
          </section>
        ) : null}

        <section className="mt-10" aria-labelledby="updates">
          <SectionHeader
            id="updates"
            title="进度更新"
            description="创作者公开的筹资与交付进展。"
          />
          <div className="mt-4 space-y-4">
            {project.updates.length > 0 ? (
              project.updates.map((update) => (
                <Card key={update.updateId} className="space-y-1.5">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="text-body font-medium text-text-primary">{update.title}</h3>
                    <span className="shrink-0 text-label text-text-tertiary">
                      {formatDateTime(update.publishTime)}
                    </span>
                  </div>
                  <p className="whitespace-pre-line text-caption leading-relaxed text-text-secondary">
                    {update.content}
                  </p>
                </Card>
              ))
            ) : (
              <EmptyState title="暂无进度更新" description="有新的进展时，创作者会在这里同步。" />
            )}
          </div>
        </section>

        <section className="mt-10" aria-labelledby="faq">
          <SectionHeader id="faq" title="常见问题" />
          <div className="mt-4 divide-y divide-divider rounded-md border border-border bg-bg-card">
            {sortedFaq.length > 0 ? (
              sortedFaq.map((faq) => (
                <details key={faq.faqId} className="group px-4 py-3">
                  <summary className="flex cursor-pointer list-none items-center justify-between gap-3 text-body font-medium text-text-primary">
                    {faq.question}
                    <span aria-hidden className="text-text-tertiary transition-transform group-open:rotate-45">＋</span>
                  </summary>
                  <p className="mt-2 whitespace-pre-line text-caption leading-relaxed text-text-secondary">
                    {faq.answer}
                  </p>
                </details>
              ))
            ) : (
              <p className="px-4 py-6 text-caption text-text-tertiary">暂无常见问题。</p>
            )}
          </div>
        </section>

        <section className="mt-10 space-y-3" aria-labelledby="risk">
          <SectionHeader id="risk" title="风险与退款" />
          {project.riskNote ? (
            <Alert tone="warning" title="风险提示">
              <p className="whitespace-pre-line">{project.riskNote}</p>
            </Alert>
          ) : null}
          <Alert tone="info" title="退款规则">
            <p className="whitespace-pre-line">
              {project.refundRule || '退款规则以企划页说明为准。'}
              {project.refundDays > 0 ? `支持后 ${project.refundDays} 天内可按规则申请退款。` : ''}
            </p>
          </Alert>
        </section>

        {project.creator ? (
          <section className="mt-10" aria-labelledby="creator">
            <SectionHeader id="creator" title="关于创作者" />
            <Card className="mt-4 space-y-3">
              <div className="flex items-center gap-3">
                <span className="flex size-11 items-center justify-center rounded-full bg-brand-subtle text-subtitle font-bold text-brand">
                  {project.creator.displayName.slice(0, 1)}
                </span>
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="truncate text-body font-medium text-text-primary">
                      {project.creator.displayName}
                    </p>
                    {project.creator.isVerified ? <Tag tone="brand">已认证</Tag> : null}
                  </div>
                  {project.creator.bio ? (
                    <p className="mt-0.5 line-clamp-2 text-caption text-text-tertiary">{project.creator.bio}</p>
                  ) : null}
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2 border-t border-divider pt-3">
                <Stat label="发起企划" value={project.creator.projectNum} align="center" />
                <Stat label="成功企划" value={project.creator.successNum} align="center" />
                <Stat
                  label="履约率"
                  value={formatPercent(permilleToPercent(project.creator.fulfillmentRate))}
                  align="center"
                />
              </div>
            </Card>
          </section>
        ) : null}
      </article>

      <aside className="hidden lg:block">
        <div className="sticky top-24">
          <SupportCard project={project} />
        </div>
      </aside>
    </div>
  );
}
