import { describe, expect, it, vi } from 'vitest';

import { ApiClient, type FetchLike } from './client';
import { ApiError } from './errors';

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'content-type': 'application/json; charset=utf-8', ...init.headers },
    ...init,
  });
}

function createClient(fetchImpl: FetchLike, extra: Partial<ConstructorParameters<typeof ApiClient>[0]> = {}) {
  return new ApiClient({
    baseUrl: 'http://127.0.0.1:8098',
    fetchImpl,
    ...extra,
  });
}

describe('ApiClient.request', () => {
  it('规范化 baseUrl 并丢弃空查询参数', async () => {
    const fetchImpl = vi.fn<FetchLike>(async () => jsonResponse([]));
    const client = createClient(fetchImpl);

    await client.getProjects({ page: 2, pageSize: 20, categoryId: undefined, sort: undefined });

    expect(fetchImpl).toHaveBeenCalledTimes(1);
    const [url] = fetchImpl.mock.calls[0]!;
    expect(url).toBe('http://127.0.0.1:8098/v1/projects?page=2&pageSize=20');
  });

  it('解析裸数组响应', async () => {
    const fetchImpl = vi.fn<FetchLike>(async () =>
      jsonResponse([{ projectId: 1, title: '示例企划' }]),
    );
    const client = createClient(fetchImpl);

    const items = await client.getHotProjects({ pageSize: 5 });

    expect(items).toHaveLength(1);
    expect(items[0]?.title).toBe('示例企划');
  });

  it('把服务端错误体映射为 ApiError', async () => {
    const fetchImpl = vi.fn<FetchLike>(async () =>
      new Response(JSON.stringify({ code: 'NOT_FOUND', message: '企划不存在', traceId: 'trace-1' }), {
        status: 404,
        headers: { 'content-type': 'application/json', 'x-trace-id': 'trace-1' },
      }),
    );

    const error = await createClient(fetchImpl)
      .getProject(42)
      .catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    const apiError = error as ApiError;
    expect(apiError.status).toBe(404);
    expect(apiError.code).toBe('NOT_FOUND');
    expect(apiError.message).toBe('企划不存在');
    expect(apiError.traceId).toBe('trace-1');
    expect(apiError.kind).toBe('http');
  });

  it('网络异常转换为 network 类型错误', async () => {
    const fetchImpl = vi.fn<FetchLike>(async () => {
      throw new TypeError('fetch failed');
    });

    const error = (await createClient(fetchImpl)
      .getBanners()
      .catch((err: unknown) => err)) as ApiError;

    expect(error).toBeInstanceOf(ApiError);
    expect(error.kind).toBe('network');
    expect(error.isRetryable).toBe(true);
  });

  it('超时中断请求', async () => {
    const fetchImpl = vi.fn<FetchLike>(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            reject(new DOMException('Aborted', 'AbortError'));
          });
        }),
    );

    const error = (await createClient(fetchImpl, { timeoutMs: 5 })
      .getCategories()
      .catch((err: unknown) => err)) as ApiError;

    expect(error.kind).toBe('timeout');
    expect(error.isTimeout).toBe(true);
  });

  it('捕获 Set-Cookie 中的 psSession', async () => {
    const onSession = vi.fn();
    const fetchImpl = vi.fn<FetchLike>(
      async () =>
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: {
            'content-type': 'application/json',
            'set-cookie': 'psSession=tok-123; Path=/; HttpOnly; SameSite=Lax',
          },
        }),
    );

    await createClient(fetchImpl, { onSession }).get<{ ok: boolean }>('v1/user');

    expect(onSession).toHaveBeenCalledWith('tok-123');
  });

  it('服务端请求注入 Cookie 与 UA', async () => {
    const fetchImpl = vi.fn<FetchLike>(async () => jsonResponse({}));
    const client = createClient(fetchImpl, { sessionToken: 'tok-9', deviceId: 'dev-1' });

    await client.get('v1/user');

    const [, init] = fetchImpl.mock.calls[0]!;
    const headers = init?.headers as Record<string, string>;
    expect(headers.Cookie).toBe('psSession=tok-9');
    expect(headers.dvid).toBe('dev-1');
    expect(headers['User-Agent']).toContain('PlanstarWeb');
  });

  it('榜单 limit 上限被夹到 50', async () => {
    const fetchImpl = vi.fn<FetchLike>(async () =>
      jsonResponse({ type: 'hot', generatedAt: 0, items: [] }),
    );

    await createClient(fetchImpl).getRankings({ type: 'hot', limit: 200 });

    const [url] = fetchImpl.mock.calls[0]!;
    expect(url).toContain('limit=50');
  });

  it('成功响应不是 JSON 时抛解析错误', async () => {
    const fetchImpl = vi.fn<FetchLike>(
      async () => new Response('<html></html>', { status: 200, headers: { 'content-type': 'text/html' } }),
    );

    const error = (await createClient(fetchImpl)
      .getBanners()
      .catch((err: unknown) => err)) as ApiError;

    expect(error.kind).toBe('parse');
  });
});
