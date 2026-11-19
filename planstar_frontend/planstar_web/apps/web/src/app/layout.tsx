import type { Metadata, Viewport } from 'next';

import { SiteFooter } from '@/components/site-footer';
import { SiteHeader } from '@/components/site-header';
import { site } from '@/lib/site';
import './globals.css';

export const metadata: Metadata = {
  metadataBase: new URL(site.url),
  title: {
    default: `${site.name} · ${site.slogan}`,
    template: `%s · ${site.name}`,
  },
  description: site.description,
  applicationName: site.name,
  keywords: ['Planstar', '企划星球', '企划', '众筹', '创作者', '回报档位', '线下演出'],
  icons: {
    icon: '/planstar-icon.png',
    apple: '/planstar-icon.png',
  },
  openGraph: {
    type: 'website',
    locale: 'zh_CN',
    siteName: site.name,
    title: `${site.name} · ${site.slogan}`,
    description: site.description,
    url: '/',
  },
  robots: {
    index: true,
    follow: true,
  },
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#F9FAFC' },
    { media: '(prefers-color-scheme: dark)', color: '#131417' },
  ],
};

/** 首屏前同步主题，避免暗色用户看到白屏闪烁。 */
const themeScript = `(function(){try{var t=localStorage.getItem('planstar-theme');var d=window.matchMedia('(prefers-color-scheme: dark)').matches;if(t==='dark'||(!t&&d)){document.documentElement.classList.add('dark');document.documentElement.style.colorScheme='dark'}}catch(e){}})();`;

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="flex min-h-dvh flex-col">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[60] focus:rounded-sm focus:bg-bg-card focus:px-4 focus:py-2 focus:text-body focus:text-brand"
        >
          跳转到主要内容
        </a>
        <SiteHeader />
        <main id="main" className="flex-1">
          {children}
        </main>
        <SiteFooter />
      </body>
    </html>
  );
}
