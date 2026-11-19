import { cookies } from 'next/headers';

import { ApiClient, toUserMessage } from '@planstar/api-client';

/** 服务端默认后端地址（本地联调端口与 Kratos 配置一致）。 */
const DEFAULT_API_BASE = 'http://127.0.0.1:8098/';

function resolveTimeout(): number {
  const parsed = Number(process.env.PLANSTAR_API_TIMEOUT_MS ?? 15_000);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 15_000;
}

/**
 * 创建服务端 API 客户端。
 *
 * 每个请求单独创建，并把入站 Cookie 中的 `psSession` 转发给后端，
 * 避免跨请求串号（Next 服务端渲染是多用户共享进程）。
 */
export async function createServerApiClient(): Promise<ApiClient> {
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get('psSession')?.value;
  const baseUrl = process.env.PLANSTAR_API_BASE ?? DEFAULT_API_BASE;

  return new ApiClient({
    baseUrl,
    sessionToken,
    timeoutMs: resolveTimeout(),
    deviceId: cookieStore.get('dvid')?.value,
  });
}

export interface Loaded<T> {
  data: T | null;
  error: unknown;
  /** 可直接展示的中文错误文案；成功时为 null。 */
  message: string | null;
}

/**
 * 把单个接口调用包成不会抛错的结果对象。
 *
 * 首页等聚合页各楼层独立降级：一个楼层失败不影响其他楼层渲染，
 * 也不会把整页打成 500。
 */
export async function load<T>(promise: Promise<T>): Promise<Loaded<T>> {
  try {
    return { data: await promise, error: null, message: null };
  } catch (error) {
    return { data: null, error, message: toUserMessage(error) };
  }
}
