import type { Admin } from "@/services";

/**
 * @see https://umijs.org/docs/max/access#access
 * */
export default function access(
  initialState: { currentUser?: Admin } | undefined,
) {
  const { currentUser } = initialState ?? {};
  const access = currentUser?.access?.toLowerCase();
  return {
    canAdmin: access === "admin",
    canOperate: access === "admin" || access === "operator",
    canView: access === "admin" || access === "operator" || access === "viewer",
  };
}
