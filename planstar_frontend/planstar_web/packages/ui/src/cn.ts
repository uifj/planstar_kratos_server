export type ClassValue = string | number | false | null | undefined;

/** 极简 className 合并，避免为三个字符串引入额外依赖。 */
export function cn(...values: ClassValue[]): string {
  return values.filter((value) => typeof value === 'string' && value.length > 0).join(' ');
}
