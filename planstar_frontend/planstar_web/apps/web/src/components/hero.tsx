import Link from 'next/link';

import type { Banner } from '@planstar/api-client';
import { buttonClass, Cover, Tag } from '@planstar/ui';

import { site } from '@/lib/site';
import { GuaranteeBar } from './guarantee-bar';

export interface HeroProps {
  banners?: Banner[];
}

/** 首页首屏：品牌主张 + 主行动 + 运营 banner / 品牌面板。 */
export function Hero({ banners = [] }: HeroProps) {
  const banner = banners[0];

  return (
    <section className="relative overflow-hidden border-b border-divider bg-bg-card">
      <div aria-hidden className="ps-hero-grid pointer-events-none absolute inset-0 opacity-70" />
      <div
        aria-hidden
        className="pointer-events-none absolute -right-24 -top-32 size-[420px] rounded-full bg-brand-subtle opacity-70 blur-3xl"
      />

      <div className="ps-container relative grid items-center gap-10 py-14 md:grid-cols-[1.1fr_0.9fr] md:py-20">
        <div>
          <Tag tone="brand" size="md">
            PLANSTAR · 企划共创平台
          </Tag>
          <h1 className="mt-5 text-[36px] font-bold leading-[1.2] tracking-tight text-text-primary md:text-[46px]">
            {site.slogan}
          </h1>
          <p className="mt-4 max-w-xl text-body leading-relaxed text-text-secondary md:text-subtitle">
            发起企划、公开筹资进度、按回报档位支持。Planstar
            把创作者的想法、支持者的期待与每一次交付记录在同一条时间线上。
          </p>

          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Link href="/projects" className={buttonClass({ size: 'lg' })}>
              发现企划
            </Link>
            <Link href="/rankings" className={buttonClass({ size: 'lg', variant: 'secondary' })}>
              查看榜单
            </Link>
          </div>

          <GuaranteeBar className="mt-8 flex max-w-xl flex-wrap items-center gap-x-3 gap-y-1.5 rounded-sm border border-divider bg-bg-card px-4 py-2.5" />
        </div>

        <div className="relative">
          {banner ? (
            <Link
              href={banner.showId ? `/projects/${banner.showId}` : '/projects'}
              className="block overflow-hidden rounded-lg border border-border bg-bg-card"
            >
              <Cover
                src={banner.image}
                alt={banner.title || 'Planstar 精选企划'}
                fallback={banner.title || 'Planstar 精选企划'}
                variant="poster"
                ratio="4/3"
                priority
              />
              <div className="border-t border-divider px-4 py-3">
                <p className="text-label uppercase tracking-[0.18em] text-brand">精选推荐</p>
                <p className="mt-1 line-clamp-1 text-body font-medium text-text-primary">
                  {banner.title || '本期精选企划'}
                </p>
              </div>
            </Link>
          ) : (
            <div className="relative overflow-hidden rounded-lg border border-border bg-bg-card p-7">
              <p className="text-label uppercase tracking-[0.24em] text-text-tertiary">PLANSTAR</p>
              <p className="mt-4 text-[28px] font-bold leading-tight text-text-primary">
                从想法到交付
                <br />
                每一步都看得见
              </p>
              <div className="mt-7 space-y-3">
                {[
                  ['发起企划', '目标金额、回报档位与交付计划一次说清'],
                  ['公开进度', '筹资、达成、更新与评价全程留痕'],
                  ['按约交付', '支持者按档位获得回报，平台跟踪履约'],
                ].map(([title, desc], index) => (
                  <div key={title} className="flex gap-3 rounded-sm bg-bg-subtle px-3.5 py-3">
                    <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-brand-subtle text-caption font-bold text-brand">
                      {index + 1}
                    </span>
                    <div>
                      <p className="text-caption font-medium text-text-primary">{title}</p>
                      <p className="mt-0.5 text-label leading-relaxed text-text-tertiary">{desc}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
