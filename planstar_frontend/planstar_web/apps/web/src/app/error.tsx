'use client';

import Link from 'next/link';
import { useEffect } from 'react';

import { buttonClass, EmptyState } from '@planstar/ui';

/** 路由级错误边界：渲染异常时给出重试入口，不暴露堆栈。 */
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error('[planstar-web] 页面渲染失败', error);
  }, [error]);

  return (
    <div className="ps-container py-20">
      <EmptyState
        icon="!"
        title="页面出了点问题"
        description="请重试；如果问题持续出现，请稍后再访问。"
        action={
          <div className="flex flex-wrap justify-center gap-3">
            <button type="button" onClick={reset} className={buttonClass({ size: 'sm' })}>
              重新加载
            </button>
            <Link href="/" className={buttonClass({ size: 'sm', variant: 'secondary' })}>
              返回首页
            </Link>
          </div>
        }
      />
    </div>
  );
}
