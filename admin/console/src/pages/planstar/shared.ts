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

/** 金额字段（分）转成人民币文案。protojson 把 int64 序列化成字符串，两种都收。 */
export const formatCents = (cents?: number | string) => {
  if (cents === undefined || cents === null || cents === '') return '-';
  const value = Number(cents);
  return Number.isFinite(value) ? `¥${(value / 100).toFixed(2)}` : '-';
};

/** 达成率以千分数返回，1000 表示 100%。 */
export const formatPermille = (permille?: number) =>
  typeof permille === 'number' ? `${(permille / 10).toFixed(1)}%` : '-';

/** 手机号脱敏，同时保留原值供复制。 */
export const maskMobile = (mobile?: string) => {
  if (!mobile) return '-';
  if (mobile.length < 7) return mobile;
  return `${mobile.slice(0, 3)}****${mobile.slice(-4)}`;
};

/** useIntl() 的最小结构类型，避免直接依赖 react-intl 的类型导出。 */
type StatusIntl = { formatMessage: (descriptor: { id: string }) => string };

/** 状态枚举的单一事实源：文案 id + ProTable 徽章色。列表与详情抽屉共用。 */
export type StatusMeta = {
  id: string;
  status: 'Default' | 'Success' | 'Processing' | 'Error' | 'Warning';
};

export const IDEA_PUBLISH_STATUS: Record<number, StatusMeta> = {
  0: { id: 'pages.planstar.projects.publish.draft', status: 'Default' },
  1: { id: 'pages.planstar.projects.publish.published', status: 'Success' },
  2: { id: 'pages.planstar.projects.publish.reviewing', status: 'Processing' },
  3: { id: 'pages.planstar.projects.publish.rejected', status: 'Error' },
};

export const GOODS_STATUS: Record<number, StatusMeta> = {
  1: { id: 'pages.planstar.goods.status.on', status: 'Success' },
  0: { id: 'pages.planstar.goods.status.off', status: 'Default' },
};

export const FEED_STATUS: Record<number, StatusMeta> = {
  1: { id: 'pages.planstar.feeds.status.normal', status: 'Success' },
  0: { id: 'pages.planstar.feeds.status.hidden', status: 'Default' },
};

export const ORDER_STATUS: Record<number, StatusMeta> = {
  1: { id: 'pages.planstar.orders.status.init', status: 'Default' },
  2: { id: 'pages.planstar.orders.status.paying', status: 'Processing' },
  3: { id: 'pages.planstar.orders.status.waitingForTickets', status: 'Processing' },
  4: { id: 'pages.planstar.orders.status.gettingTickets', status: 'Processing' },
  5: { id: 'pages.planstar.orders.status.finished', status: 'Success' },
  6: { id: 'pages.planstar.orders.status.canceled', status: 'Error' },
  7: { id: 'pages.planstar.orders.status.refunded', status: 'Default' },
};

export const PAY_ORDER_STATUS: Record<string, StatusMeta> = {
  pending: { id: 'pages.planstar.payOrders.status.pending', status: 'Default' },
  success: { id: 'pages.planstar.payOrders.status.success', status: 'Success' },
  failed: { id: 'pages.planstar.payOrders.status.failed', status: 'Error' },
  expired: { id: 'pages.planstar.payOrders.status.expired', status: 'Warning' },
};

export const REFUND_STATUS: Record<string, StatusMeta> = {
  applied: { id: 'pages.planstar.refunds.status.applied', status: 'Warning' },
  processing: { id: 'pages.planstar.refunds.status.processing', status: 'Processing' },
  success: { id: 'pages.planstar.refunds.status.success', status: 'Success' },
  rejected: { id: 'pages.planstar.refunds.status.rejected', status: 'Default' },
};

/** 把状态映射转成 ProTable valueEnum（保留徽章色）。 */
export function statusValueEnum(
  intl: StatusIntl,
  map: Record<number | string, StatusMeta>,
) {
  return Object.fromEntries(
    Object.entries(map).map(([key, meta]) => [
      key,
      { text: intl.formatMessage({ id: meta.id }), status: meta.status },
    ]),
  );
}

/** 详情抽屉里把状态值转成本地化文案；缺省回退 '-'，未知值原样显示。 */
export function statusText(
  intl: StatusIntl,
  map: Record<number | string, StatusMeta>,
  value?: number | string,
) {
  if (value === undefined || value === null || value === '') return '-';
  const meta = map[value];
  return meta ? intl.formatMessage({ id: meta.id }) : String(value);
}
