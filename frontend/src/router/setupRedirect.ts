/**
 * 登录后的默认落地页：管理员默认打开账号管理，普通用户进入用户控制台。
 */
export function resolveDefaultLandingPath(isAdmin: boolean): string {
  return isAdmin ? '/admin/accounts' : '/dashboard'
}

export function resolveCompletedSetupRedirectPath(isAuthenticated: boolean, isAdmin: boolean): string {
  if (!isAuthenticated) {
    return '/login'
  }

  return resolveDefaultLandingPath(isAdmin)
}
