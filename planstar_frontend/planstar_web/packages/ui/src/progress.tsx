import { cn } from './cn';

export interface ProgressProps {
  /** 0–100（超过 100 时轨道填满）。 */
  value: number;
  className?: string;
  height?: 4 | 6 | 8;
  /** 无障碍名称。 */
  label?: string;
}

/** 进度条：品牌色填充 + 细描边轨道，无渐变。 */
export function Progress({ value, className, height = 6, label }: ProgressProps) {
  const clamped = Number.isFinite(value) ? Math.min(Math.max(value, 0), 100) : 0;
  const heightClass = height === 4 ? 'h-1' : height === 8 ? 'h-2' : 'h-1.5';

  return (
    <div
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(clamped)}
      aria-label={label}
      className={cn('w-full overflow-hidden rounded-full bg-bg-subtle', heightClass, className)}
    >
      <div
        className="h-full rounded-full bg-brand transition-[width] duration-300"
        style={{ width: `${clamped}%` }}
      />
    </div>
  );
}
