import type { ButtonHTMLAttributes, ReactNode } from 'react';

import { cn } from './cn';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'subtle';
export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonStyleOptions {
  variant?: ButtonVariant;
  size?: ButtonSize;
  block?: boolean;
  className?: string;
}

const variantClass: Record<ButtonVariant, string> = {
  // 主按钮：药丸形（20px），与 Flutter FilledButton 一致。
  primary:
    'bg-brand text-on-brand hover:bg-brand-pressed active:bg-brand-pressed disabled:bg-brand-disabled disabled:text-text-disabled rounded-xl',
  // 次按钮：8px 圆角 + 细描边。
  secondary:
    'bg-bg-card text-brand border border-border hover:border-brand hover:text-brand-pressed disabled:text-text-disabled disabled:border-border rounded-sm',
  ghost: 'text-brand hover:bg-brand-subtle rounded-sm',
  subtle: 'bg-brand-subtle text-brand hover:bg-brand-disabled rounded-sm',
};

const sizeClass: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-caption gap-1',
  md: 'h-10 px-4 text-body gap-1.5',
  lg: 'h-11 px-6 text-body gap-2',
};

/** 生成按钮类名，供 `<Link>` 等非 button 元素复用。 */
export function buttonClass(options: ButtonStyleOptions = {}): string {
  const { variant = 'primary', size = 'md', block = false, className } = options;
  return cn(
    'inline-flex select-none items-center justify-center whitespace-nowrap font-medium',
    'transition-colors duration-150 outline-none',
    'focus-visible:ring-2 focus-visible:ring-brand/40 focus-visible:ring-offset-1 focus-visible:ring-offset-bg-page',
    'disabled:cursor-not-allowed disabled:opacity-100',
    variantClass[variant],
    sizeClass[size],
    block && 'w-full',
    className,
  );
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement>, ButtonStyleOptions {
  children: ReactNode;
}

export function Button({
  variant = 'primary',
  size = 'md',
  block = false,
  className,
  type = 'button',
  children,
  ...rest
}: ButtonProps) {
  return (
    <button type={type} className={buttonClass({ variant, size, block, className })} {...rest}>
      {children}
    </button>
  );
}
