/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

/**
 * Next user-console homepage. `/dashboard` redirects here; both forms must
 * drop the sidebar shell so the first paint is the standalone console page.
 */
export function isNextConsoleHomePath(pathname: string): boolean {
  return pathname === '/dashboard' || pathname === '/dashboard/overview'
}

/**
 * Usage-logs routes share the standalone Next chrome but fill the remaining
 * viewport like classic list pages, instead of the landing max-width measure.
 */
export function isNextUsageLogsPath(pathname: string): boolean {
  return pathname === '/usage-logs' || pathname.startsWith('/usage-logs/')
}

/**
 * Authenticated routes that keep the Next standalone chrome (top bar, no
 * sidebar). The homepage, personal center, wallet, and usage logs share
 * this shell; other console pages stay on the classic sidebar layout.
 */
export function isNextStandaloneShellPath(pathname: string): boolean {
  return (
    isNextConsoleHomePath(pathname) ||
    pathname === '/profile' ||
    pathname === '/wallet' ||
    isNextUsageLogsPath(pathname)
  )
}

/**
 * URL prefixes that live under `_authenticated`. Public, auth and legal pages
 * are intentionally absent so they cannot flip the Next console chrome.
 */
const NEXT_CONSOLE_PATH_PREFIXES = [
  '/dashboard',
  '/profile',
  '/wallet',
  '/usage-logs',
  '/keys',
  '/playground',
  '/chat',
  '/chat2link',
  '/channels',
  '/channel-monitoring-settings',
  '/models',
  '/users',
  '/redemption-codes',
  '/subscriptions',
  '/system-info',
  '/system-settings',
  '/errors',
] as const

export function isNextConsolePath(pathname: string): boolean {
  return NEXT_CONSOLE_PATH_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`)
  )
}

/**
 * Path used to choose standalone vs sidebar chrome while a navigation is
 * still pending. Public paths never decide the shell: keep the console path
 * that is still on screen, otherwise take the console destination, otherwise
 * stay on the standalone homepage so the classic sidebar cannot flash.
 */
export function resolveNextConsoleShellPathname(
  pendingPathname: string,
  renderedPathname?: string
): string {
  const rendered = renderedPathname ?? pendingPathname
  if (isNextConsolePath(rendered)) return rendered
  if (isNextConsolePath(pendingPathname)) return pendingPathname
  return '/dashboard/overview'
}
