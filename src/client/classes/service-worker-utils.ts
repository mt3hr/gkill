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

/**
 * gkill API の応答が成功か。HTTP ok かつ errors が空。
 * 成功時 errors は null で返る（omitempty が無い）ので、長さを見る前に null を通すこと。
 * Does not consume the body (reads via clone).
 */
export async function is_successful_gkill_response(response: Response): Promise<boolean> {
  if (!response.ok) return false
  try {
    const json = await response.clone().json()
    if (json.errors && json.errors.length > 0) return false
  } catch {
    return false
  }
  return true
}

/**
 * Validate whether a Response should be cached. Does not consume the body (reads via clone).
 *
 * 本文のパースは1回だけにすること。以前は is_successful_gkill_response を呼んだうえで
 * もう一度 clone().json() していたため、SWでキャッシュする1行ごとの応答
 * (get_kyou / get_kmemo / get_tags_by_id ...) が SW内で2回 + ページで1回の計3回パースされていた。
 * 一覧のスクロールでは行数ぶん飛ぶので、SWスレッドの占有としてそのまま体感に出る。
 */
export async function should_cache_response(response: Response, check_histories: boolean): Promise<boolean> {
  if (!response.ok) return false
  let json: Record<string, unknown>
  try {
    json = await response.clone().json()
  } catch {
    return false
  }
  const errors = json.errors
  if (Array.isArray(errors) && errors.length > 0) return false
  if (check_histories) {
    for (const key of Object.keys(json)) {
      if (key.endsWith('_histories')) {
        if (Array.isArray(json[key]) && json[key].length === 0) return false
        break
      }
    }
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
