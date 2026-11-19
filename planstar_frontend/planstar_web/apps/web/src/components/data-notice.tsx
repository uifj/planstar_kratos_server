import { Alert } from '@planstar/ui';

export interface DataNoticeProps {
  message: string;
  title?: string;
  className?: string;
}

/**
 * 数据加载失败提示。
 *
 * 聚合页单个楼层失败时只降级这一块，不阻断整页；
 * 文案里给出可执行动作（刷新页面），不暴露后端错误细节。
 */
export function DataNotice({ message, title = '内容暂时无法加载', className }: DataNoticeProps) {
  return (
    <Alert tone="warning" title={title} className={className}>
      <p>{message}</p>
      <p className="mt-1 text-text-tertiary">
        请稍后<a href="" className="mx-1 text-brand underline-offset-2 hover:underline">刷新页面</a>重试。
      </p>
    </Alert>
  );
}
