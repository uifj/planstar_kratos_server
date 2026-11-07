import { createAdminServiceClient } from "@/services/kratos/admin/v1";
import { createConsoleServiceClient } from "@/services/planstar/admin/v1";
import { request } from "@umijs/max";

type Request = {
  path: string;
  method: string;
  body: string | null;
};

type RequestMeta = {
  service: string;
  method: string;
};

function requestHandler(
  { path, method, body }: Request,
  meta?: RequestMeta,
) {
  // Use protojson so proto semantics (oneof, well-known types like Timestamp)
  // are preserved in both directions. kratos selects the codec from the
  // Content-Type / Accept subtype, so both headers must name protojson.
  const headers: Record<string, string> = {
    Accept: "application/protojson",
  };
  // Generated clients pass a body only on POST / PUT / PATCH.
  if (body) {
    headers["Content-Type"] = "application/protojson";
  }
  return request("/" + path, {
    method: method,
    data: body,
    headers: headers,
    // Auth is a cookie (psAdminToken); keep it on every request, including
    // same-origin deployments behind a reverse proxy.
    credentials: "include",
    withCredentials: true,
    // 企划审核的黑名单 400 由弹窗自行 message.error，避免全局提示重复一次。
    skipErrorHandler:
      meta?.service === "ConsoleService" && meta.method === "ReviewProject",
  });
}

// Generated clients are stateless closures over the handler, so one instance
// per service is enough. Add a resource by adding one line here; call sites
// keep importing `services` unchanged.
export const services = {
  admin: createAdminServiceClient(requestHandler),
  console: createConsoleServiceClient(requestHandler),
};

/**
 * The structured error body the server returns (kratos `errors.Status` in its
 * protojson form). This is a data shape, not an Error subclass: narrow it with
 * `asServiceError` rather than `instanceof`.
 */
export type ServiceError = {
  code?: number;
  reason?: string;
  message?: string;
  metadata?: Record<string, string>;
};

/**
 * Read the structured error body out of whatever `request` rejected with.
 * Returns undefined when the failure carries no structured body — a gateway
 * plain-text 502, a proxy timeout, an aborted request.
 */
export function asServiceError(error: unknown): ServiceError | undefined {
  const data = (error as { response?: { data?: unknown } })?.response?.data;
  return data && typeof data === "object" ? (data as ServiceError) : undefined;
}

// Re-export DTOs so pages import types and clients from the same place. If a
// second domain ever collides on a type name, switch to namespaced exports
// (`export * as adminV1 from "@/services/kratos/admin/v1"`).
export type * from "@/services/kratos/admin/v1";
export type * from "@/services/planstar/admin/v1";
