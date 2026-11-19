'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState } from 'react';

import { buttonClass, cn } from '@planstar/ui';

import { navItems, site } from '@/lib/site';
import { ThemeToggle } from './theme-toggle';

function isActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(`${href}/`);
}

/** 全站顶部导航：品牌、主导航、主题切换与主 CTA。 */
export function SiteHeader() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50 border-b border-divider bg-bg-page/85 backdrop-blur-md">
      <div className="ps-container flex h-16 items-center justify-between gap-4">
        <Link href="/" className="flex shrink-0 items-center gap-2.5" aria-label={`${site.name} 首页`}>
          <Image
            src="/planstar-icon.png"
            alt=""
            width={32}
            height={32}
            priority
            className="size-8 rounded-sm object-contain"
          />
          <span className="flex flex-col leading-none">
            <span className="text-subtitle font-bold tracking-wide text-text-primary">PLANSTAR</span>
            <span className="mt-1 text-label tracking-[0.18em] text-text-tertiary">企划星球</span>
          </span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex" aria-label="主导航">
          {navItems.map((item) => {
            const active = isActive(pathname, item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'rounded-sm px-3 py-2 text-body transition-colors',
                  active
                    ? 'bg-brand-subtle font-medium text-brand'
                    : 'text-text-secondary hover:bg-bg-subtle hover:text-text-primary',
                )}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="flex items-center gap-2">
          <ThemeToggle />
          <Link href="/projects" className={buttonClass({ size: 'sm', className: 'hidden sm:inline-flex' })}>
            浏览企划
          </Link>
          <button
            type="button"
            className="inline-flex size-9 items-center justify-center rounded-sm border border-border bg-bg-card text-text-secondary md:hidden"
            aria-label={open ? '关闭菜单' : '打开菜单'}
            aria-expanded={open}
            onClick={() => setOpen((value) => !value)}
          >
            <svg viewBox="0 0 24 24" className="size-[18px]" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
              {open ? (
                <path d="M6 6l12 12M18 6L6 18" strokeLinecap="round" />
              ) : (
                <path d="M4 7h16M4 12h16M4 17h16" strokeLinecap="round" />
              )}
            </svg>
          </button>
        </div>
      </div>

      {open ? (
        <nav className="border-t border-divider bg-bg-card md:hidden" aria-label="移动端导航">
          <ul className="ps-container flex flex-col py-2">
            {navItems.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={() => setOpen(false)}
                  className={cn(
                    'block rounded-sm px-3 py-3 text-body',
                    isActive(pathname, item.href)
                      ? 'bg-brand-subtle font-medium text-brand'
                      : 'text-text-secondary',
                  )}
                >
                  {item.label}
                  {item.description ? (
                    <span className="mt-0.5 block text-caption text-text-tertiary">{item.description}</span>
                  ) : null}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      ) : null}
    </header>
  );
}
