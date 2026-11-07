import { useRef } from 'react';

/** 每页条数。后端 page_size 上限 200，20 与列表默认值一致。 */
export const PAGE_SIZE = 20;

/**
 * 后端列表接口是游标分页（nextPageToken），ProTable 只认页码。这里按页缓存
 * 游标：第 N 页的请求使用第 N-1 页返回的 token；筛选条件变化时清空缓存。
 *
 * 接口不返回总数，total 只用于让分页器知道「还有下一页」：当前页序号 × 页大小
 * 加上本页条数，再在存在 nextPageToken 时补 1，下一页按钮就会亮起。
 */
export function useCursorPager(pageSize = PAGE_SIZE) {
  const tokens = useRef<string[]>([]);
  const signatureRef = useRef('');

  return {
    pageToken(current: number, signature: string) {
      if (signature !== signatureRef.current) {
        signatureRef.current = signature;
        tokens.current = [];
      }
      return current <= 1 ? undefined : tokens.current[current - 1];
    },
    total(current: number, count: number, nextPageToken?: string) {
      if (nextPageToken) {
        tokens.current[current] = nextPageToken;
      } else {
        tokens.current = tokens.current.slice(0, current);
      }
      return (current - 1) * pageSize + count + (nextPageToken ? 1 : 0);
    },
  };
}

/**
 * 生成客户端用 `if (request.status)` 拼 query，0 会被省略。后端 status /
 * publish_status 是 optional int32，0（禁用用户、草稿企划）是合法过滤值；
 * 传字符串 "0" 可绕过生成代码的真值判断，服务端仍按 int32 解析。
 */
export const optionalIntFilter = (value?: number) =>
  value === 0 ? ('0' as unknown as number) : value;

/** 金额字段（分）转成人民币文案。 */
export const formatCents = (cents?: number) =>
  typeof cents === 'number' ? `¥${(cents / 100).toFixed(2)}` : '-';

/** 达成率以千分数返回，1000 表示 100%。 */
export const formatPermille = (permille?: number) =>
  typeof permille === 'number' ? `${(permille / 10).toFixed(1)}%` : '-';

/** 手机号脱敏，同时保留原值供复制。 */
export const maskMobile = (mobile?: string) => {
  if (!mobile) return '-';
  if (mobile.length < 7) return mobile;
  return `${mobile.slice(0, 3)}****${mobile.slice(-4)}`;
};
