import path from 'node:path';

import type { NextConfig } from 'next';

/**
 * Planstar 官网 Next.js 配置。
 *
 * - 后端跨域白名单为空（仅同源），浏览器侧统一走 `/api/backend/*` 反向代理；
 * - 服务端渲染直接请求 `PLANSTAR_API_BASE`，不经过自身代理；
 * - 远程图片域名由运营配置，使用原生 <img>，不开启 next/image 远程域名白名单；
 * - 生产镜像使用 standalone 产物（见 planstar_web/Dockerfile）。
 */
const rawApiBase = process.env.PLANSTAR_API_BASE ?? 'http://127.0.0.1:8098/';
const apiBase = rawApiBase.endsWith('/') ? rawApiBase : `${rawApiBase}/`;

const nextConfig: NextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  // 只打包运行时必需文件；pnpm 构建时 cwd 为 apps/web，因此上溯两级到 monorepo 根
  output: 'standalone',
  outputFileTracingRoot: path.join(process.cwd(), '..', '..'),
  transpilePackages: ['@planstar/ui', '@planstar/api-client', '@planstar/design-tokens'],
  experimental: {
    optimizePackageImports: ['@planstar/ui', '@planstar/api-client'],
  },
  async rewrites() {
    return [
      {
        source: '/api/backend/:path*',
        destination: `${apiBase}:path*`,
      },
    ];
  },
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'X-Frame-Options', value: 'SAMEORIGIN' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
        ],
      },
    ];
  },
};

export default nextConfig;
