import Image from 'next/image';
import Link from 'next/link';

import { navItems, site } from '@/lib/site';

/** 全站页脚：品牌说明、导航与平台保障。 */
export function SiteFooter() {
  return (
    <footer className="mt-auto border-t border-divider bg-bg-card">
      <div className="ps-container grid gap-10 py-12 md:grid-cols-[1.4fr_1fr_1fr]">
        <div>
          <div className="flex items-center gap-2.5">
            <Image src="/planstar-icon.png" alt="" width={28} height={28} className="size-7 object-contain" />
            <span className="text-subtitle font-bold text-text-primary">{site.name}</span>
          </div>
          <p className="mt-3 max-w-md text-caption leading-relaxed text-text-tertiary">
            {site.description}
          </p>
        </div>

        <div>
          <h2 className="text-body font-medium text-text-primary">浏览</h2>
          <ul className="mt-3 space-y-2">
            {navItems.map((item) => (
              <li key={item.href}>
                <Link href={item.href} className="text-caption text-text-tertiary transition-colors hover:text-brand">
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h2 className="text-body font-medium text-text-primary">平台保障</h2>
          <ul className="mt-3 space-y-2 text-caption text-text-tertiary">
            <li>100% 履约与交付跟踪</li>
            <li>未达成按规则退款</li>
            <li>支持者与创作者双向评价</li>
          </ul>
        </div>
      </div>

      <div className="border-t border-divider">
        <div className="ps-container flex flex-col gap-2 py-5 text-label text-text-disabled sm:flex-row sm:items-center sm:justify-between">
          <p>{site.copyright}</p>
          <p>{site.slogan}</p>
        </div>
      </div>
    </footer>
  );
}
