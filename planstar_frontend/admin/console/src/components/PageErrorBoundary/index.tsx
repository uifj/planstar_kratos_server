import { getIntl } from '@umijs/max';
import { Button, Result } from 'antd';
import { Component, type ErrorInfo, type ReactNode } from 'react';

type Props = { children: ReactNode };
type State = { error: Error | null };

/**
 * 页面级错误边界（ADR-0010 P2⑨）：兜住路由子树里任何未捕获的渲染/生命周期异常，
 * 避免一个页面崩溃把整个控制台拖白屏。捕获后给出「重新加载」出口。
 *
 * 它与 requestErrorConfig 的分工：那边管「请求失败」，这里管「渲染失败」。
 */
export default class PageErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // 上报走前端监控/控制台；这里不重抛，避免把错误边界自身也炸掉。
    console.error('[PageErrorBoundary]', error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      const intl = getIntl();
      return (
        <Result
          status="error"
          title={intl.formatMessage({ id: 'pages.error.renderFailed.title' })}
          subTitle={intl.formatMessage({
            id: 'pages.error.renderFailed.subTitle',
          })}
          extra={
            <Button
              type="primary"
              onClick={() => window.location.reload()}
            >
              {intl.formatMessage({ id: 'pages.error.reload' })}
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}

const CHUNK_ERROR_RE =
  /Loading chunk|ChunkLoadError|dynamically imported module|Failed to fetch dynamically imported module/i;

/**
 * chunk 加载失败兜底（ADR-0010 P2⑨）：路由懒加载的 chunk 在发布后失效（文件名
 * hash 变化）时，动态 import 会抛错。这里监听全局 error 事件，命中 chunk 失败就
 * 整页刷新一次（带防抖，避免同一时刻多个 chunk 都失败时重复刷新）。
 */
export function installChunkErrorRecovery() {
  let reloading = false;
  const onError = (event: ErrorEvent) => {
    if (reloading) return;
    const message = event.error instanceof Error ? event.error.message : '';
    if (!CHUNK_ERROR_RE.test(message)) return;
    reloading = true;
    window.location.reload();
  };
  window.addEventListener('error', onError);
}
