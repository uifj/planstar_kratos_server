import { guarantees } from '@/lib/site';

/**
 * 服务保障条。
 *
 * 对应 Flutter 首页 26dp 白底细描边条：一排 12sp 文案，
 * 项间用 1px 分隔线。Web 端按内容自适应高度并保留同样的视觉节奏。
 */
export function GuaranteeBar({ className }: { className?: string }) {
  return (
    <div
      className={
        className ??
        'flex flex-wrap items-center justify-center gap-x-3 gap-y-1.5 rounded-sm border border-divider bg-bg-card px-4 py-2'
      }
    >
      {guarantees.map((item, index) => (
        <span key={item} className="flex items-center gap-3">
          {index > 0 ? <span aria-hidden className="h-2.5 w-px bg-border" /> : null}
          <span className="text-caption text-text-primary">{item}</span>
        </span>
      ))}
    </div>
  );
}
