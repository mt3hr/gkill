/**
 * Utility functions extracted from serviceWorker.ts for testability.
 * ServiceWorker globals (self, caches, etc.) cause side effects on import,
 * so these pure functions are separated.
 */

/**
 * Cache name for POST responses keyed by a Kyou/target id (/cache/api/{data_type}/{id}).
 * Shared by serviceWorker.ts (put/match) and delete-gkill-cache.ts (delete) — the two must never drift.
 */
export const KYOU_CACHE_NAME = 'gkill-post-kyou-cache'

/** Cache name for POST responses that have no id (/cache/api/{data_type}). */
export const CONFIG_CACHE_NAME = 'gkill-post-config-cache'

/** Validate whether a Response should be cached. Does not consume the body (reads via clone). */
export async function should_cache_response(response: Response, check_histories: boolean): Promise<boolean> {
  if (!response.ok) return false
  try {
    const json = await response.clone().json()
    if (json.errors && json.errors.length > 0) return false
    if (check_histories) {
      for (const key of Object.keys(json)) {
        if (key.endsWith('_histories')) {
          if (Array.isArray(json[key]) && json[key].length === 0) return false
          break
        }
      }
    }
  } catch {
    return false
  }
  return true
}

/** Parse a loose boolean value: true/1/yes/y, false/0/no/n (case-insensitive, trimmed). */
export function parse_bool_loose(value: unknown): boolean {
  if (typeof value === "boolean") return value
  if (typeof value === "number") return value !== 0
  if (typeof value === "string") {
    const v = value.trim().toLowerCase()
    if (["true", "1", "yes", "y"].includes(v)) return true
    if (["false", "0", "no", "n"].includes(v)) return false
  }
  throw new SyntaxError(`Boolean expected, got ${JSON.stringify(value)}`)
}
