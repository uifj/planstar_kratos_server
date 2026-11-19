/** 服务端统一错误体：`{ code, message, traceId }`。 */
export interface ApiErrorPayload {
  code?: string;
  message?: string;
  traceId?: string;
  [key: string]: unknown;
}

export interface ApiErrorInit {
  status: number;
  code: string;
  message: string;
  traceId?: string;
  requestId?: string;
  url?: string;
  cause?: unknown;
  kind?: 'http' | 'network' | 'timeout' | 'parse';
}

/**
 * 统一 API 异常。
 *
 * - `kind === 'http'`：服务端已返回非 2xx，`status` / `code` 可用；
 * - `kind === 'network'`：DNS / 连接失败等，无法拿到响应；
 * - `kind === 'timeout'`：超过 `timeoutMs`；
 * - `kind === 'parse'`：响应不是合法 JSON。
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly traceId?: string;
  readonly requestId?: string;
  readonly url?: string;
  readonly kind: NonNullable<ApiErrorInit['kind']>;

  constructor(init: ApiErrorInit) {
    super(init.message, { cause: init.cause });
    this.name = 'ApiError';
    this.status = init.status;
    this.code = init.code;
    this.traceId = init.traceId;
    this.requestId = init.requestId;
    this.url = init.url;
    this.kind = init.kind ?? 'http';
  }

  get isNetworkError(): boolean {
    return this.kind === 'network';
  }

  get isTimeout(): boolean {
    return this.kind === 'timeout';
  }

  /** 是否属于「重试可能有帮助」的错误。 */
  get isRetryable(): boolean {
    return this.kind === 'network' || this.kind === 'timeout' || this.status >= 500 || this.status === 429;
  }

  toJSON(): Record<string, unknown> {
    return {
      name: this.name,
      status: this.status,
      code: this.code,
      message: this.message,
      traceId: this.traceId,
      requestId: this.requestId,
      url: this.url,
      kind: this.kind,
    };
  }
}

/** HTTP 状态码 → 面向用户的中文提示。 */
export function messageForStatus(status: number): string {
  if (status === 400) return '请求参数有误，请稍后重试';
  if (status === 401) return '登录已过期，请重新登录';
  if (status === 403) return '没有访问权限';
  if (status === 404) return '内容不存在或已下线';
  if (status === 409) return '数据已变更，请刷新后重试';
  if (status === 413) return '提交的内容过大';
  if (status === 429) return '访问过于频繁，请稍后再试';
  if (status >= 500) return '服务暂时不可用，请稍后重试';
  return '网络开小差了，请稍后重试';
}

/**
 * 把任意异常转换成可展示的中文文案。
 *
 * 页面只消费这个函数，避免各页面各写一套 `error.message`。
 */
export function toUserMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.kind === 'timeout') return '请求超时，请检查网络后重试';
    if (error.kind === 'network') return '无法连接到服务，请稍后重试';
    if (error.kind === 'parse') return '服务返回了无法识别的数据';
    return error.message || messageForStatus(error.status);
  }
  if (error instanceof Error && error.message) return error.message;
  return '发生未知错误，请稍后重试';
}

/** 是否为「内容不存在」类错误，详情页据此渲染 404 空状态。 */
export function isNotFound(error: unknown): boolean {
  return error instanceof ApiError && error.status === 404;
}
