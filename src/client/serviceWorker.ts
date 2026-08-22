// 編集前に読む: .claude/skills/gkill-client-foundation/SKILL.md（この領域の不変条件の正本）
/// <reference lib="webworker" />
import delete_gkill_kyou_cache from './classes/delete-gkill-cache';
import { clientsClaim } from 'workbox-core'
import { cleanupOutdatedCaches, precacheAndRoute, createHandlerBoundToURL, } from 'workbox-precaching'
import { registerRoute, NavigationRoute } from 'workbox-routing'
import { CacheFirst } from 'workbox-strategies'
import { should_cache_response, should_cache_for_session, is_successful_gkill_response, parse_bool_loose, KYOU_CACHE_NAME, CONFIG_CACHE_NAME } from './classes/service-worker-utils';
import {
  SHARE_FORCE_SAVE_FORM_KEY,
  SHARE_TARGET_PATH,
  append_share_ledger,
  decide_share_save_target,
  find_duplicated_share_entry,
  read_share_ledger,
  write_share_ledger,
  type SharedPayload,
  type ShareLedgerEntry,
} from './classes/share-target-dedup';

declare let clients: Clients;
declare let self: ServiceWorkerGlobalScope

export default null

// M-9: 現在のセッションIDを cookieStore から取る（無い環境では undefined）。
// アカウント切替後に旧セッションの飛行中応答が別利用者へキャッシュされるのを防ぐ判定に使う。
async function get_current_session_id_for_cache(): Promise<string | undefined> {
  try {
    if (typeof cookieStore !== 'undefined') {
      const cookie = await cookieStore.get('gkill_session_id')
      return cookie?.value ?? undefined
    }
  } catch { /* cookieStore 非対応。フォールバックする */ }
  return undefined
}

self.skipWaiting()
clientsClaim()

cleanupOutdatedCaches()

// 版が変わったらKyou系の派生キャッシュは丸ごと捨てる。
// このキャッシュの個別削除は「検索直前に get_updated_datas_by_time で更新IDを引く」
// 経路に頼っているが、あの表のUpdatedTimeは前進しないので、一度取りこぼした更新は
// 二度と通知されず古い応答が焼き付いたまま残る。
// cleanupOutdatedCaches() はWorkboxのprecacheしか掃除しないのでここで消す。
self.addEventListener('activate', event => {
  event.waitUntil(caches.delete(KYOU_CACHE_NAME))
})

precacheAndRoute(self.__WB_MANIFEST, {
  directoryIndex: null as unknown as string,
})

// precache から外した遅延チャンク (mermaid系など) のランタイムキャッシュ。
// vite.config.ts の precacheGlobIgnores で precache 対象外にしているぶんをここで拾う。
// ファイル名に content hash が付いていて中身が変わることは無いので CacheFirst でよい。
// precache 済みのアセットは上の precacheAndRoute が先にルートを持つのでここには来ない。
registerRoute(
  ({ url, request }) => request.destination === 'script' && url.pathname.startsWith('/assets/'),
  new CacheFirst({ cacheName: 'gkill-lazy-chunk-cache' }),
)

// SPA の app-shell (index.html) フォールバック。ただし / と /api/ /files/ /zip_cache/ /resources/manual/ は除外
registerRoute(
  new NavigationRoute(createHandlerBoundToURL('index.html'), {
    denylist: [
      /^\/$/,        // "/" は除外
      /^\/api\/.*/,  // "/api/..." は除外
      /^\/files\/.*/,  // "/files/..." は除外
      /^\/zip_cache\/.*/,  // "/zip_cache/..." は除外（ZIP展開キャッシュ）
      /^\/resources\/manual\/.*/,  // "/resources/manual/..." は除外（ヘルプHTML）
      /^\/share-target$/,  // "/share-target" は除外（下の専用ハンドラが respondWith する）
    ],
  }),
)

self.addEventListener('push', async function (event: PushEvent) {
  if (!event.data) return;
  const data = event.data.json()
  if (data.is_notification) {
    const title = 'gkill'
    const options = {
      body: data.content,
      requireInteraction: true,
      data: data,
      timestamp: Math.floor(new Date(data.time as string).getTime())
    }
    event.waitUntil(self.registration.showNotification(title, options))
  } else if (data.is_updated_data_notify) {
    await delete_gkill_kyou_cache(data.id)
  }
})

self.addEventListener('notificationclick', function (event) {
  const data = event.notification.data
  event.notification.close()
  event.waitUntil(
    clients.openWindow(data.url)
  )
})

self.addEventListener('fetch', (event: FetchEvent) => {
  const { request } = event

  const url = new URL(request.url)
  if (request.method === 'POST' && (
    // Kyou系。
    // ここを増やしたら delete-gkill-cache.ts の data_types にも足すこと。
    // 足さないと更新後も古いキャッシュが返り続ける
    url.pathname === '/api/get_kyou' ||
    url.pathname === '/api/get_kmemo' ||
    url.pathname === '/api/get_kc' ||
    url.pathname === '/api/get_urlog' ||
    url.pathname === '/api/get_nlog' ||
    url.pathname === '/api/get_timeis' ||
    url.pathname === '/api/get_mi' ||
    url.pathname === '/api/get_lantana' ||
    url.pathname === '/api/get_rekyou' ||
    url.pathname === '/api/get_mirekyou' ||
    url.pathname === '/api/get_git_commit_log' ||
    url.pathname === '/api/get_idf_kyou' ||
    url.pathname === '/api/get_tags_by_id' ||
    url.pathname === '/api/get_texts_by_id' ||
    url.pathname === '/api/get_gkill_notifications_by_id')) {
    event.respondWith(
      (async () => {
        try {
          const req_clone1 = request.clone()
          const req_clone2 = request.clone()

          const body = await req_clone1.json()
          const force_reget = parse_bool_loose(body.force_reget)
          const id = body.target_id ? body.target_id : body.id

          const data_type = new URL(request.url).pathname.replace('/api/get_', '')
          const cache_key = `/cache/api/${data_type}/${id}`

          const kyou_cache = await caches.open(KYOU_CACHE_NAME)
          if (!force_reget) {
            const cached = await kyou_cache.match(cache_key)
            if (cached) return cached
          }

          const response = await fetch(req_clone2)
          if (await should_cache_response(response, true)) {
            if (should_cache_for_session(body.session_id, await get_current_session_id_for_cache())) {
              kyou_cache.put(cache_key, response.clone())
            }
          }
          return response

        } catch (err: unknown) {
          if (err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            return Response.error()
          } else {
            // abort以外はエラー出力する
            console.error('[SW] fetch handler error', err)
            try { return await fetch(request.clone()) } catch { return Response.error() }
          }
        }
      })()
    )
  } else if (request.method === 'POST' &&
    url.pathname === '/api/get_plugin_content_html') {
    event.respondWith(
      (async () => {
        try {
          const req_clone1 = request.clone()
          const req_clone2 = request.clone()

          const body = await req_clone1.json()
          const force_reget = parse_bool_loose(body.force_reget)
          const id = body.kyou_id

          const cache_key = `/cache/api/plugin_content_html/${id}`

          const kyou_cache = await caches.open(KYOU_CACHE_NAME)
          if (!force_reget) {
            const cached = await kyou_cache.match(cache_key)
            if (cached) return cached
          }

          const response = await fetch(req_clone2)
          if (await should_cache_response(response, true)) {
            if (should_cache_for_session(body.session_id, await get_current_session_id_for_cache())) {
              kyou_cache.put(cache_key, response.clone())
            }
          }
          return response

        } catch (err: unknown) {
          if (err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            return Response.error()
          } else {
            // abort以外はエラー出力する
            console.error('[SW] fetch handler error', err)
            try { return await fetch(request.clone()) } catch { return Response.error() }
          }
        }
      })()
    )
  } else if (request.method === 'POST' && (
    // ApplicationConfig系
    url.pathname === '/api/get_all_rep_names' ||
    url.pathname === '/api/get_all_tag_names' ||
    url.pathname === '/api/get_mi_board_list' ||
    url.pathname === '/api/get_application_config')) {
    event.respondWith(
      (async () => {
        try {
          const req_clone0 = request.clone()
          const req_clone1 = request.clone()

          const body = await req_clone0.json()
          const force_reget = parse_bool_loose(body.force_reget)

          const data_type = new URL(request.url).pathname.replace('/api/get_', '')
          const cache_key = `/cache/api/${data_type}`

          const config_cache = await caches.open(CONFIG_CACHE_NAME)
          if (!force_reget) {
            const cached = await config_cache.match(cache_key)
            if (cached) return cached
          }

          const response = await fetch(req_clone1)
          if (await should_cache_response(response, false)) {
            if (should_cache_for_session(body.session_id, await get_current_session_id_for_cache())) {
              config_cache.put(cache_key, response.clone())
            }
          }
          return response
        } catch (err: unknown) {
          if (err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            return Response.error()
          } else {
            // abort以外はエラー出力する
            console.error('[SW] fetch handler error', err)
            try { return await fetch(request.clone()) } catch { return Response.error() }
          }
        }
      })()
    )
  }
})

/** 共有された内容を保存する。保存できたときだけ true を返す（台帳へ載せてよいかの判断に使う）。 */
async function save_shared_payload(payload: SharedPayload): Promise<boolean> {
  try {
    const target = decide_share_save_target(payload)
    if (!target) {
      return false
    }

    // Get session ID via Cookie Store API (available in service workers)
    let session_id = ""
    if (typeof cookieStore !== 'undefined') {
      const cookie = await cookieStore.get('gkill_session_id')
      if (cookie && cookie.value) {
        session_id = cookie.value
      }
    }

    const now = new Date(Date.now())

    // Get device/user info from application config
    const config_res = await fetch('/api/get_application_config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ session_id, locale_name: 'en' }),
    })
    const config_json = await config_res.json()
    const app_config = config_json.application_config ?? {}
    const device: string = app_config.device ?? ""
    const user_id: string = app_config.user_id ?? ""

    const make_kyou_base = () => ({
      is_deleted: false,
      id: crypto.randomUUID(),
      rep_name: "",
      related_time: now,
      data_type: "",
      create_time: now,
      create_app: "gkill_share",
      create_device: device,
      create_user: user_id,
      update_time: now,
      update_app: "gkill_share",
      update_device: device,
      update_user: user_id,
    })

    if (target.kind === 'urlog') {
      const res = await fetch('/api/add_urlog', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          session_id,
          locale_name: 'en',
          tx_id: null,
          want_response_kyou: false,
          added_kyou: null,
          urlog: {
            ...make_kyou_base(),
            url: target.url,
            title: target.title,
            description: "",
            favicon_image: "",
            thumbnail_image: "",
          },
        }),
      })
      return await is_successful_gkill_response(res)
    }

    const res = await fetch('/api/add_kmemo', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({
        session_id,
        locale_name: 'en',
        tx_id: null,
        want_response_kyou: false,
        added_kyou: null,
        kmemo: {
          ...make_kyou_base(),
          content: target.content,
        },
      }),
    })
    return await is_successful_gkill_response(res)
  } catch (e) {
    console.error('[SW] share-target error:', e)
    return false
  }
}

self.addEventListener('fetch', event => {
  const req = event.request;
  if (req.method === 'POST' &&
    new URL(req.url).pathname === SHARE_TARGET_PATH) {

    event.respondWith((async () => {
      let form: FormData | null = null
      try {
        form = await req.formData()
      } catch (e) {
        console.error('[SW] share-target form parse error:', e)
      }
      const payload: SharedPayload = {
        title: (form?.get('title') as string | null) ?? "",
        text: (form?.get('text') as string | null) ?? "",
        url: (form?.get('url') as string | null) ?? "",
      }
      // 重複確認のうえで「それでも保存する」を押された経路。台帳の照会を飛ばす
      const is_force = form?.get(SHARE_FORCE_SAVE_FORM_KEY) != null

      const now = Date.now()
      const ledger = await read_share_ledger()

      // Androidはタスク復帰で同じ共有インテントを再配送する。届く内容は初回と同じなので、
      // 保存済みの内容を覚えておくことでしか二重保存を止められない
      const duplicated = is_force ? null : find_duplicated_share_entry(ledger, payload, now)
      if (duplicated) {
        return Response.redirect('/saihate?share_result=duplicate&share_entry_id=' + encodeURIComponent(duplicated.id), 303)
      }

      const is_saved = await save_shared_payload(payload)
      if (is_saved) {
        // 保存できたときだけ載せる。失敗を載せると、保存できていないのに次の共有が弾かれる。
        // 強制保存では既存の行の時刻を進める（次の再配送もそこから期間ぶん弾く）
        const existing = find_duplicated_share_entry(ledger, payload, now)
        const entry: ShareLedgerEntry = {
          id: existing ? existing.id : crypto.randomUUID(),
          saved_at: now,
          payload,
        }
        // 応答を返す前に必ず書き切る。SWは応答後すぐ止まりうる
        await write_share_ledger(append_share_ledger(ledger, entry, now))
      }

      if (is_force) {
        return new Response(JSON.stringify({ is_saved }), {
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return Response.redirect('/saihate?is_saved=' + is_saved, 303);
    })());
  }
});
