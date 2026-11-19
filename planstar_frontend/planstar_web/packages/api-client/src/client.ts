import { ApiError, messageForStatus, type ApiErrorPayload } from './errors';
import type {
  Banner,
  Category,
  PageParams,
  ProjectDetail,
  ProjectFaq,
  ProjectListParams,
  ProjectRanking,
  ProjectSummary,
  ProjectUpdate,
  RankingParams,
  ReviewPage,
  ReviewParams,
  RewardTier,
} from './types';

export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export type QueryValue = string | number | boolean | undefined | null;

export interface ApiClientOptions {
  /** 必须以 `/` 结尾，例如 `https://api.planstar.top/`。 */
  baseUrl: string;
  fetchImpl?: FetchLike;
  timeoutMs?: number;
  /** 设备标识，服务端按 `dvid` 头读取；缺省时由调用方按需注入。 */
  deviceId?: string;
  /** 会话令牌，服务端渲染时从 Cookie 中取出后注入。 */
  sessionToken?: string;
  /** 额外请求头。 */
  defaultHeaders?: Record<string, string>;
  /** 会话令牌变化回调（登录后服务端通过 Set-Cookie 下发）。 */
  onSession?: (token: string) => void;
}

export interface RequestOptions extends Omit<RequestInit, 'body' | 'headers'> {
  query?: Record<string, QueryValue>;
  body?: unknown;
  headers?: Record<string, string>;
  timeoutMs?: number;
}

export type ProjectUpdateParams = PageParams;

const DEFAULT_TIMEOUT_MS = 15_000;

function trimTrailingSlash(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

function normalizeBaseUrl(baseUrl: string): string {
  const trimmed = baseUrl.trim();
  if (!trimmed) {
    throw new Error('ApiClient: baseUrl 不能为空');
  }
  return trimmed.endsWith('/') ? trimmed : `${trimmed}/`;
}

function buildQuery(query?: Record<string, QueryValue>): string {
  if (!query) return '';
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue;
    search.set(key, String(value));
  }
  const qs = search.toString();
  return qs ? `?${qs}` : '';
}

/** 从 `Set-Cookie` 中提取 `psSession`，用于服务端转发登录态。 */
function extractSession(response: Response): string | undefined {
  const direct = response.headers.get('psSession');
  if (direct) return direct;

  const cookies = response.headers.get('set-cookie');
  if (!cookies) return undefined;
  const match = /(?:^|,\s*)psSession=([^;,\s]+)/.exec(cookies);
  return match?.[1];
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException
    ? error.name === 'AbortError'
    : error instanceof Error && error.name === 'AbortError';
}

/**
 * Planstar 接口客户端。
 *
 * 设计要点：
 * - 裸 JSON 响应（无 `{code,data}` 信封），非 2xx 才读错误体；
 * - 超时、网络错误、解析错误统一抛 {@link ApiError}；
 * - 服务端渲染时通过 `sessionToken` 注入 `Cookie: psSession=…`，
 *   浏览器端走同源 `/api/backend` 反向代理，由 Next 转发 Cookie；
 * - 浏览器环境禁止手工设置 `Cookie` / `User-Agent`，这里按运行时分流。
 */
export class ApiClient {
  private readonly baseUrl: string;
  private readonly fetchImpl: FetchLike;
  private readonly timeoutMs: number;
  private readonly deviceId?: string;
  private readonly sessionToken?: string;
  private readonly defaultHeaders: Record<string, string>;
  private readonly onSession?: (token: string) => void;

  constructor(options: ApiClientOptions) {
    this.baseUrl = normalizeBaseUrl(options.baseUrl);
    this.fetchImpl = options.fetchImpl ?? ((input, init) => fetch(input, init));
    this.timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    this.deviceId = options.deviceId;
    this.sessionToken = options.sessionToken;
    this.defaultHeaders = options.defaultHeaders ?? {};
    this.onSession = options.onSession;
  }

  /** 复制一个带指定会话令牌的实例（服务端按请求创建，避免串号）。 */
  withSession(sessionToken?: string): ApiClient {
    return new ApiClient({
      baseUrl: this.baseUrl,
      fetchImpl: this.fetchImpl,
      timeoutMs: this.timeoutMs,
      deviceId: this.deviceId,
      sessionToken,
      defaultHeaders: this.defaultHeaders,
      onSession: this.onSession,
    });
  }

  /** 复制一个带设备标识的实例。 */
  withDevice(deviceId?: string): ApiClient {
    return new ApiClient({
      baseUrl: this.baseUrl,
      fetchImpl: this.fetchImpl,
      timeoutMs: this.timeoutMs,
      deviceId,
      sessionToken: this.sessionToken,
      defaultHeaders: this.defaultHeaders,
      onSession: this.onSession,
    });
  }

  /** 发起一次请求并解析裸 JSON。 */
  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { query, body, headers, timeoutMs, ...init } = options;
    const relativePath = path.replace(/^\/+/, '');
    const url = `${this.baseUrl}${relativePath}${buildQuery(query)}`;

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs ?? this.timeoutMs);
    const externalSignal = init.signal;
    const abortFromOutside = () => controller.abort();
    externalSignal?.addEventListener('abort', abortFromOutside, { once: true });

    const isServer = typeof window === 'undefined';
    const requestHeaders: Record<string, string> = {
      Accept: 'application/json',
      ...this.defaultHeaders,
      ...headers,
    };

    if (isServer) {
      // 浏览器把这两个头视为 forbidden header，禁止手工设置。
      requestHeaders['User-Agent'] ??= 'PlanstarWeb/0.1 (+https://planstar.top)';
      if (this.sessionToken) {
        requestHeaders.Cookie = `psSession=${this.sessionToken}`;
      }
    }
    if (this.deviceId) {
      requestHeaders.dvid = this.deviceId;
    }

    let payload: BodyInit | undefined;
    if (body !== undefined) {
      if (typeof body === 'string' || body instanceof FormData || body instanceof Blob) {
        payload = body;
      } else {
        payload = JSON.stringify(body);
        requestHeaders['Content-Type'] ??= 'application/json';
      }
    }

    let response: Response;
    try {
      response = await this.fetchImpl(url, {
        ...init,
        headers: requestHeaders,
        body: payload,
        signal: controller.signal,
        credentials: init.credentials ?? 'include',
      });
    } catch (error) {
      if (isAbortError(error)) {
        throw new ApiError({
          status: 0,
          code: 'TIMEOUT',
          message: '请求超时，请检查网络后重试',
          url,
          cause: error,
          kind: 'timeout',
        });
      }
      throw new ApiError({
        status: 0,
        code: 'NETWORK_ERROR',
        message: '无法连接到服务，请稍后重试',
        url,
        cause: error,
        kind: 'network',
      });
    } finally {
      clearTimeout(timeout);
      externalSignal?.removeEventListener('abort', abortFromOutside);
    }

    const session = extractSession(response);
    if (session && this.onSession) {
      this.onSession(session);
    }

    const requestId = response.headers.get('x-trace-id') ?? response.headers.get('X-Trace-Id') ?? undefined;
    const text = await response.text();
    const isJson = (response.headers.get('content-type') ?? '').includes('application/json');

    if (!response.ok) {
      const payloadError = parseErrorPayload(text, isJson);
      throw new ApiError({
        status: response.status,
        code: payloadError.code ?? `HTTP_${response.status}`,
        message: payloadError.message ?? messageForStatus(response.status),
        traceId: payloadError.traceId ?? requestId,
        requestId,
        url,
        kind: 'http',
      });
    }

    if (!text) {
      return null as T;
    }

    if (!isJson) {
      throw new ApiError({
        status: response.status,
        code: 'PARSE_ERROR',
        message: '服务返回了无法识别的数据',
        requestId,
        url,
        kind: 'parse',
      });
    }

    try {
      return JSON.parse(text) as T;
    } catch (error) {
      throw new ApiError({
        status: response.status,
        code: 'PARSE_ERROR',
        message: '服务返回了无法识别的数据',
        requestId,
        url,
        cause: error,
        kind: 'parse',
      });
    }
  }

  get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, { ...options, method: 'GET' });
  }

  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, { ...options, method: 'POST', body });
  }

  // ── 企划域端点（契约 7.7 / 8.10）─────────────────────────────────────────

  /** `GET v1/home/banners` → 裸数组。 */
  getBanners(): Promise<Banner[]> {
    return this.get<Banner[]>('v1/home/banners');
  }

  /** `GET v1/categories` → 裸数组。 */
  getCategories(): Promise<Category[]> {
    return this.get<Category[]>('v1/categories');
  }

  /** `GET v1/projects/hot?page&pageSize` → 裸数组。 */
  getHotProjects(params: PageParams = {}): Promise<ProjectSummary[]> {
    return this.get<ProjectSummary[]>('v1/projects/hot', {
      query: { page: params.page, pageSize: params.pageSize },
    });
  }

  /** `GET v1/projects?page&pageSize&categoryId&fundingStatus&sort` → 裸数组。 */
  getProjects(params: ProjectListParams = {}): Promise<ProjectSummary[]> {
    return this.get<ProjectSummary[]>('v1/projects', {
      query: {
        page: params.page,
        pageSize: params.pageSize,
        categoryId: params.categoryId,
        fundingStatus: params.fundingStatus,
        sort: params.sort,
      },
    });
  }

  /** `GET v1/projects/{id}` → 详情对象。 */
  getProject(id: number | string): Promise<ProjectDetail> {
    return this.get<ProjectDetail>(`v1/projects/${encodeURIComponent(String(id))}`);
  }

  /** `GET v1/projects/{id}/reward-tiers` → 裸数组。 */
  getRewardTiers(id: number | string): Promise<RewardTier[]> {
    return this.get<RewardTier[]>(`v1/projects/${encodeURIComponent(String(id))}/reward-tiers`);
  }

  /** `GET v1/projects/{id}/updates?page&pageSize` → 裸数组。 */
  getProjectUpdates(id: number | string, params: ProjectUpdateParams = {}): Promise<ProjectUpdate[]> {
    return this.get<ProjectUpdate[]>(`v1/projects/${encodeURIComponent(String(id))}/updates`, {
      query: { page: params.page, pageSize: params.pageSize },
    });
  }

  /** `GET v1/projects/{id}/faqs` → 裸数组。 */
  getProjectFaqs(id: number | string): Promise<ProjectFaq[]> {
    return this.get<ProjectFaq[]>(`v1/projects/${encodeURIComponent(String(id))}/faqs`);
  }

  /** `GET v1/projects/{id}/reviews?sort&onlyShow&page&pageSize` → 评价分页对象。 */
  getProjectReviews(id: number | string, params: ReviewParams = {}): Promise<ReviewPage> {
    return this.get<ReviewPage>(`v1/projects/${encodeURIComponent(String(id))}/reviews`, {
      query: {
        sort: params.sort,
        onlyShow: params.onlyShow === undefined ? undefined : params.onlyShow ? 1 : 0,
        page: params.page,
        pageSize: params.pageSize,
      },
    });
  }

  /** `GET v1/rankings?type&categoryId&limit` → 榜单对象（limit ≤ 50）。 */
  getRankings(params: RankingParams = {}): Promise<ProjectRanking> {
    return this.get<ProjectRanking>('v1/rankings', {
      query: {
        type: params.type,
        categoryId: params.categoryId,
        limit: params.limit === undefined ? undefined : Math.min(params.limit, 50),
      },
    });
  }
}

function parseErrorPayload(text: string, isJson: boolean): ApiErrorPayload {
  if (!text) return {};
  if (!isJson) return { message: text.slice(0, 200) };
  try {
    const parsed: unknown = JSON.parse(text);
    return parsed && typeof parsed === 'object' ? (parsed as ApiErrorPayload) : {};
  } catch {
    return {};
  }
}

/** 去掉末尾斜杠的辅助导出，供反向代理配置复用。 */
export { trimTrailingSlash };
