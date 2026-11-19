import Link from 'next/link';

import type { Metadata } from 'next';

import { buttonClass, Card, SectionHeader, Tag } from '@planstar/ui';

import { GuaranteeBar } from '@/components/guarantee-bar';
import { site } from '@/lib/site';

export const metadata: Metadata = {
  title: '关于我们',
  description:
    'Planstar 企划星球希望把企划的目标、进度、回报与交付放在同一条公开时间线上，让创作者与支持者建立可验证的信任。',
};

const steps = [
  {
    index: '01',
    title: '发起企划',
    description:
      '创作者说明企划内容、目标金额、回报档位与交付计划。平台审核后公开展示，支持者在支持前即可看到完整信息。',
  },
  {
    index: '02',
    title: '公开筹资',
    description:
      '企划页实时展示已筹金额、达成率、支持人数与剩余时间。创作者通过进度更新同步制作与筹备情况。',
  },
  {
    index: '03',
    title: '按约交付',
    description:
      '达成目标后进入交付阶段。支持者按所选档位获得回报，平台跟踪履约情况，并保留评价与售后记录。',
  },
];

const principles = [
  {
    title: '信息透明',
    description: '目标、进度、档位、时间与风险提示都在企划页公开，不做隐藏条款。',
  },
  {
    title: '双向尊重',
    description: '创作者对支持者负责，支持者也理解创作的不确定性，双方通过公开记录建立信任。',
  },
  {
    title: '可追踪的履约',
    description: '每一次进度更新、交付与评价都留痕，让承诺可以被验证。',
  },
];

export default function AboutPage() {
  return (
    <>
      <section className="border-b border-divider bg-bg-card">
        <div className="ps-container py-14 md:py-20">
          <Tag tone="brand" size="md">关于 Planstar</Tag>
          <h1 className="mt-5 max-w-3xl text-[32px] font-bold leading-tight text-text-primary md:text-[40px]">
            {site.slogan}
          </h1>
          <p className="mt-4 max-w-2xl text-body leading-relaxed text-text-secondary md:text-subtitle">
            Planstar 企划星球是一个面向创作者与支持者的企划共创平台。我们希望把企划从「一句想法」到「按约交付」的全过程
            放在同一条公开时间线上，让每一个好企划都能被看见，也让每一次支持都有据可查。
          </p>
          <div className="mt-7 flex flex-wrap gap-3">
            <Link href="/projects" className={buttonClass({ size: 'lg' })}>
              发现企划
            </Link>
            <Link href="/rankings" className={buttonClass({ size: 'lg', variant: 'secondary' })}>
              查看榜单
            </Link>
          </div>
        </div>
      </section>

      <section className="ps-section">
        <div className="ps-container">
          <SectionHeader title="企划如何运转" description="从发起到交付，三个关键阶段。" />
          <div className="mt-6 grid gap-4 md:grid-cols-3">
            {steps.map((step) => (
              <Card key={step.index} className="space-y-3">
                <span className="text-title font-bold tabular-nums text-brand">{step.index}</span>
                <h3 className="text-subtitle font-bold text-text-primary">{step.title}</h3>
                <p className="text-caption leading-relaxed text-text-secondary">{step.description}</p>
              </Card>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-divider bg-bg-card">
        <div className="ps-container py-12 md:py-16">
          <SectionHeader title="平台保障" description="保障不是口号，而是写进规则的动作。" />
          <GuaranteeBar className="mt-5 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-sm border border-divider bg-bg-page px-4 py-3" />
          <div className="mt-6 grid gap-4 md:grid-cols-3">
            {principles.map((item) => (
              <div key={item.title} className="rounded-sm bg-bg-page px-4 py-4">
                <h3 className="text-body font-medium text-text-primary">{item.title}</h3>
                <p className="mt-1.5 text-caption leading-relaxed text-text-tertiary">{item.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="ps-section">
        <div className="ps-container">
          <div className="rounded-lg border border-border bg-bg-card px-6 py-10 md:px-10">
            <h2 className="text-title font-bold text-text-primary">一起把好企划做出来</h2>
            <p className="mt-2 max-w-2xl text-caption leading-relaxed text-text-tertiary">
              官网展示平台上的公开企划与榜单；发起企划、支持企划与查看订单请在 Planstar App 内完成。
            </p>
            <div className="mt-5 flex flex-wrap gap-3">
              <Link href="/projects" className={buttonClass({ size: 'md' })}>
                浏览企划
              </Link>
              <Link href="/rankings" className={buttonClass({ size: 'md', variant: 'secondary' })}>
                看看榜单
              </Link>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}
