import { cn } from './cn';

export interface CoverProps {
  src?: string | null;
  alt: string;
  /** 宽高比，默认 3:4（企划海报）。 */
  ratio?: '3/4' | '4/3' | '16/9' | '1/1';
  className?: string;
  /** 兜底文案，缺省取 alt 首字。 */
  fallback?: string;
  /**
   * 缺图时的兜底样式：
   * - `initial`：品牌淡底 + 首字，适合榜单等小尺寸缩略图；
   * - `poster`：品牌化海报（标题 + 装饰），适合卡片、详情等大尺寸封面。
   */
  variant?: 'initial' | 'poster';
  priority?: boolean;
}

const ratioClass = {
  '3/4': 'aspect-[3/4]',
  '4/3': 'aspect-[4/3]',
  '16/9': 'aspect-video',
  '1/1': 'aspect-square',
} as const;

const posterTitleClass = {
  '3/4': 'text-lg',
  '4/3': 'text-xl',
  '16/9': 'text-2xl',
  '1/1': 'text-lg',
} as const;

/**
 * 企划封面。
 *
 * 图片域名来自运营配置，无法在构建期枚举，所以使用原生 `<img>`
 * 而不是 next/image；缺图时按尺寸给出品牌化兜底，保证栅格不塌。
 */
export function Cover({
  src,
  alt,
  ratio = '3/4',
  className,
  fallback,
  variant = 'initial',
  priority = false,
}: CoverProps) {
  const title = (fallback ?? alt).trim() || 'Planstar 企划';
  const initial = title.slice(0, 1);

  return (
    <div className={cn('relative w-full overflow-hidden rounded-sm bg-brand-subtle', ratioClass[ratio], className)}>
      {src ? (
        <img
          src={src}
          alt={alt}
          loading={priority ? 'eager' : 'lazy'}
          decoding="async"
          className="absolute inset-0 h-full w-full object-cover"
        />
      ) : variant === 'poster' ? (
        <div className="absolute inset-0 flex flex-col justify-end gap-2 overflow-hidden bg-brand-subtle p-4">
          <span
            aria-hidden
            className="pointer-events-none absolute -right-8 -top-8 h-28 w-28 rounded-full border border-brand/15"
          />
          <span
            aria-hidden
            className="pointer-events-none absolute -right-1 -top-1 h-16 w-16 rounded-full border border-brand/10"
          />
          <span
            aria-hidden
            className="pointer-events-none absolute -bottom-10 -left-8 h-24 w-24 rounded-full bg-brand/5"
          />
          <p
            className={cn(
              'relative line-clamp-3 font-bold leading-snug text-brand/80',
              posterTitleClass[ratio],
            )}
          >
            {title}
          </p>
          <span className="relative flex items-center gap-2">
            <span aria-hidden className="h-px w-6 bg-brand/40" />
            <span className="text-[9px] font-medium uppercase tracking-[0.3em] text-brand/50">PLANSTAR</span>
          </span>
        </div>
      ) : (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-3xl font-bold text-brand/50">{initial}</span>
        </div>
      )}
    </div>
  );
}
