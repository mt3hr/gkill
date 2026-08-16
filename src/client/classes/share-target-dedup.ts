/**
 * Web Share Target の二重保存を防ぐ台帳。
 *
 * Android のタスク（アプリ履歴）から PWA を開き直すと、WebAPK に同じ共有インテントが
 * 再配送され、まったく同じ multipart POST /share-target が ServiceWorker にもう一度届く。
 * 届く内容は初回とビット単位で同じなので、「もう保存した共有」を内容から覚えておく以外に
 * 再配送と「利用者が意図的にもう一度共有した」を見分ける手が無い。
 *
 * 判定は全部この純関数側に置く。serviceWorker.ts は import しただけで
 * self.skipWaiting() などの副作用が走るのでテストから読めない。
 */

import { is_url } from './looks-like-url'

/**
 * 台帳を置く Cache Storage の名前。
 * Kyou 系キャッシュとは必ず分けること。serviceWorker.ts の activate は
 * KYOU_CACHE_NAME を丸ごと消すので、同居させると版が上がるたびに台帳が飛ぶ。
 */
export const SHARE_DEDUP_CACHE_NAME = 'gkill-share-dedup-cache'

/** 台帳の実体（JSON 配列）を入れている Cache のキー。 */
export const SHARE_DEDUP_LEDGER_CACHE_KEY = '/__gkill_share_dedup/ledger'

/** 同じ内容とみなす期間（24時間）。これを過ぎた共有は普通に保存する。 */
export const SHARE_DEDUP_WINDOW_MILLI_SECONDS = 24 * 60 * 60 * 1000

/** 台帳に残す最大件数。古いものから捨てる。 */
export const SHARE_DEDUP_MAX_ENTRIES = 100

/** 重複確認のうえで保存し直すときにフォームへ立てる印。 */
export const SHARE_FORCE_SAVE_FORM_KEY = 'gkill_force'

/** 共有ハンドラのパス。強制保存も同じハンドラを通す（保存の実装を2つに割らないため）。 */
export const SHARE_TARGET_PATH = '/share-target'

/** Web Share Target から届く内容。manifest の params と同じ3つ。 */
export type SharedPayload = {
    title: string
    text: string
    url: string
}

/** 台帳の1件。 */
export type ShareLedgerEntry = {
    id: string
    saved_at: number
    payload: SharedPayload
}

/** 共有内容をどの記録として保存するか。null は保存対象なし。 */
export type ShareSaveTarget =
    | { kind: 'urlog', url: string, title: string }
    | { kind: 'kmemo', content: string }
    | null

/**
 * 共有内容から保存先を決める。
 * url / title / text の順に URL らしさを見て、どれも URL でなければ本文としてメモにする。
 */
export function decide_share_save_target(payload: SharedPayload): ShareSaveTarget {
    const shared_url = payload.url
    const shared_title = payload.title
    const shared_text = payload.text

    // is_url は型述語なので、そのまま使うと否定側で string が never へ落ちて
    // このあと shared_text.split() が書けなくなる。真偽だけ受け取る
    const looks_like_url = (value: string): boolean => is_url(value)

    if (looks_like_url(shared_url)) {
        return { kind: 'urlog', url: shared_url, title: shared_title }
    }
    if (looks_like_url(shared_title)) {
        return { kind: 'urlog', url: shared_title, title: '' }
    }
    if (!shared_text) {
        return null
    }
    if (looks_like_url(shared_text)) {
        return { kind: 'urlog', url: shared_text, title: shared_title }
    }
    // AndroidのGoogleアプリだと末尾にURLが入っていることがある
    const shared_text_lines = shared_text.split('http')
    const shared_text_last_line = 'http' + shared_text_lines[shared_text_lines.length >= 2 ? shared_text_lines.length - 1 : 0]
    if (looks_like_url(shared_text_last_line)) {
        return { kind: 'urlog', url: shared_text_last_line, title: '' }
    }
    return { kind: 'kmemo', content: shared_text }
}

/** 共有内容が同じか。3フィールドが全部一致したときだけ同じ。 */
export function is_same_shared_payload(a: SharedPayload, b: SharedPayload): boolean {
    return a.title === b.title && a.text === b.text && a.url === b.url
}

/**
 * 台帳から「同じ内容をこの期間内に保存済み」の1件を探す。
 * 見つかったらそれが再配送（＝二重保存になる共有）の疑い。
 */
export function find_duplicated_share_entry(entries: Array<ShareLedgerEntry>, payload: SharedPayload, now: number): ShareLedgerEntry | null {
    for (const entry of entries) {
        if (now - entry.saved_at >= SHARE_DEDUP_WINDOW_MILLI_SECONDS) {
            continue
        }
        if (is_same_shared_payload(entry.payload, payload)) {
            return entry
        }
    }
    return null
}

/** 期限切れを落とし、新しい順に並べて上限件数で切る。 */
export function prune_share_ledger(entries: Array<ShareLedgerEntry>, now: number): Array<ShareLedgerEntry> {
    return entries
        .filter((entry) => now - entry.saved_at < SHARE_DEDUP_WINDOW_MILLI_SECONDS)
        .slice()
        .sort((a, b) => b.saved_at - a.saved_at)
        .slice(0, SHARE_DEDUP_MAX_ENTRIES)
}

/** 台帳へ1件積む。同じ id の古い行は差し替える（強制保存で saved_at を進める経路で使う）。 */
export function append_share_ledger(entries: Array<ShareLedgerEntry>, entry: ShareLedgerEntry, now: number): Array<ShareLedgerEntry> {
    const others = entries.filter((e) => e.id !== entry.id)
    return prune_share_ledger([entry, ...others], now)
}

/** 台帳を読む。Cache が無い環境でも、壊れたJSONでも落とさず空で返す。 */
export async function read_share_ledger(): Promise<Array<ShareLedgerEntry>> {
    if (typeof caches === 'undefined') {
        return []
    }
    try {
        const cache = await caches.open(SHARE_DEDUP_CACHE_NAME)
        const cached = await cache.match(SHARE_DEDUP_LEDGER_CACHE_KEY)
        if (!cached) {
            return []
        }
        const parsed: unknown = await cached.json()
        if (!Array.isArray(parsed)) {
            return []
        }
        return parsed.filter((entry): entry is ShareLedgerEntry =>
            !!entry
            && typeof entry.id === 'string'
            && typeof entry.saved_at === 'number'
            && !!entry.payload)
    } catch {
        return []
    }
}

/** 台帳を書く。応答を返す前に必ず await すること（SWはその後すぐ止まりうる）。 */
export async function write_share_ledger(entries: Array<ShareLedgerEntry>): Promise<void> {
    if (typeof caches === 'undefined') {
        return
    }
    try {
        const cache = await caches.open(SHARE_DEDUP_CACHE_NAME)
        await cache.put(SHARE_DEDUP_LEDGER_CACHE_KEY, new Response(JSON.stringify(entries), {
            headers: { 'Content-Type': 'application/json' },
        }))
    } catch (err: unknown) {
        console.error('[gkill] failed to write share dedup ledger', err)
    }
}

/** 台帳の1件を id で引く。重複確認ダイアログに中身を出すためにページ側から使う。 */
export async function find_share_ledger_entry_by_id(id: string): Promise<ShareLedgerEntry | null> {
    const entries = await read_share_ledger()
    return entries.find((entry) => entry.id === id) ?? null
}

/**
 * 重複と判定された共有を、確認のうえで保存し直す。
 * ページ側で add_urlog / add_kmemo を組み立て直すと保存が2実装に割れるので、
 * 印を1つ立てて ServiceWorker の同じハンドラを通す。
 */
export async function force_save_shared_payload(payload: SharedPayload): Promise<boolean> {
    try {
        const form = new FormData()
        form.set('title', payload.title)
        form.set('text', payload.text)
        form.set('url', payload.url)
        form.set(SHARE_FORCE_SAVE_FORM_KEY, '1')
        const res = await fetch(SHARE_TARGET_PATH, { method: 'POST', body: form })
        if (!res.ok) {
            return false
        }
        const json = await res.json()
        return json.is_saved === true
    } catch (err: unknown) {
        console.error('[gkill] failed to force save shared data', err)
        return false
    }
}
