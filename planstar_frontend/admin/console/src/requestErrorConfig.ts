import { asServiceError } from "@/services";
import { getIntl, history, type RequestConfig } from "@umijs/max";
import { message as toast } from "antd";

const loginPath = "/user/login";

// User-facing wording per error reason. A reason with no entry here falls back
// to the server message.
const reasonMessageId: Record<string, string> = {
  INVALID_CREDENTIALS: "pages.login.invalidCredentials",
  PERMISSION_DENIED: "pages.error.permissionDenied",
  PASSWORD_CHANGE_REQUIRED: "pages.passwordChange.required",
};

// 单飞（single-flight）跳转守卫：会话过期时一个页面往往有多个请求同时 401，
// 若每个都 push 登录页，history 会叠多个相同条目。这里保证并发 401 只跳一次。
// cookie 会话（psAdminToken）没有 refresh 令牌，所以「单飞」作用在跳转动作上；
// 若日后接入 9108 的 JWT access+refresh，同样的单飞模式应套用到 refresh 调用。
let loginRedirectTimer: number | undefined;

function redirectToLogin() {
  if (history.location.pathname === loginPath) return;
  if (loginRedirectTimer !== undefined) return;
  loginRedirectTimer = window.setTimeout(() => {
    loginRedirectTimer = undefined;
  }, 300);
  history.push(loginPath);
}

/**
 * Global request error handling: every failed request is reported here, so a
 * caller only has to handle its own success path. umi re-rejects afterwards, so
 * callers that need more than a toast can still catch and inspect `reason`.
 */
export const errorConfig: RequestConfig = {
  errorConfig: {
    errorHandler: (error: any, opts: any) => {
      if (opts?.skipErrorHandler) throw error;

      const serviceError = asServiceError(error);
      if (serviceError) {
        // 401 means the cookie is missing or expired. On the login page itself
        // a 401 is just a failed attempt, so don't redirect there.
        if (serviceError.code === 401) {
          redirectToLogin();
        }
        const messageId =
          serviceError.reason && reasonMessageId[serviceError.reason];
        toast.error(
          messageId
            ? getIntl().formatMessage({ id: messageId })
            : serviceError.message || "Request failed, please retry."
        );
        return;
      }

      // No structured body: a gateway error, a timeout, or an aborted request.
      if (error.response) {
        toast.error(`Response status: ${error.response.status}`);
      } else if (error.request) {
        toast.error("None response! Please retry.");
      } else {
        toast.error("Request error, please retry.");
      }
    },
  },
};
