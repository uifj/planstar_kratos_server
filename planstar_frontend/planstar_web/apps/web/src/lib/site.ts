/** 站点级常量：导航、品牌文案、对外链接。 */

export const site = {
  name: 'Planstar 企划星球',
  shortName: 'Planstar',
  slogan: '让每一个好企划，被看见',
  description:
    'Planstar 企划星球是面向创作者与支持者的企划共创平台：发起企划、公开进度、按回报档位支持，让好内容从想法走到交付。',
  url: process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000',
  copyright: `© ${new Date().getFullYear()} Planstar 企划星球`,
} as const;

export interface NavItem {
  label: string;
  href: string;
  description?: string;
}

export const navItems: NavItem[] = [
  { label: '发现企划', href: '/projects', description: '按分类浏览正在筹资的企划' },
  { label: '企划榜单', href: '/rankings', description: '热度、达成与冲刺榜' },
  { label: '关于我们', href: '/about', description: 'Planstar 的理念与保障' },
];

/** 平台级服务承诺，与 Flutter 首页保障条一致。 */
export const guarantees = ['100%履约', '未达成退款', '交付保障', '配送保障'] as const;
