import Link from 'next/link';

import { buttonClass, EmptyState } from '@planstar/ui';

export default function NotFound() {
  return (
    <div className="ps-container py-20">
      <EmptyState
        icon="404"
        title="页面不存在或已下线"
        description="你访问的企划可能已经结束展示，或者链接有误。"
        action={
          <div className="flex flex-wrap justify-center gap-3">
            <Link href="/projects" className={buttonClass({ size: 'sm' })}>
              浏览企划
            </Link>
            <Link href="/" className={buttonClass({ size: 'sm', variant: 'secondary' })}>
              返回首页
            </Link>
          </div>
        }
      />
    </div>
  );
}
