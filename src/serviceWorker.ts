/// <reference lib="webworker" />
import delete_gkill_kyou_cache from './classes/delete-gkill-cache';
import { GkillAPI } from './classes/api/gkill-api';
import { AddURLogRequest } from './classes/api/req_res/add-ur-log-request';
import { AddKmemoRequest } from './classes/api/req_res/add-kmemo-request';
import { GetGkillInfoRequest } from './classes/api/req_res/get-gkill-info-request';
import { isUrl } from './classes/looks-like-url';
import { clientsClaim } from 'workbox-core'
import { cleanupOutdatedCaches, precacheAndRoute, createHandlerBoundToURL, } from 'workbox-precaching'
import { registerRoute, NavigationRoute } from 'workbox-routing'

export default null

self.skipWaiting()
clientsClaim()
declare let clients: Clients;
declare let self: ServiceWorkerGlobalScope

cleanupOutdatedCaches()

precacheAndRoute(self.__WB_MANIFEST, {
  directoryIndex: (null as any),
})

// / は常にネットワーク（HTTP キャッシュも使わない）
registerRoute(
  ({ request, url }) => request.mode === 'navigate' && url.pathname === '/',
  async ({ event }) => {
    // no-store でブラウザHTTPキャッシュも回避
    return fetch(new Request((event as any).request, { cache: 'no-store' }))
  },
)

// SPA の app-shell (index.html) フォールバック。ただし / と /api/ は除外
registerRoute(
  new NavigationRoute(createHandlerBoundToURL('index.html'), {
    denylist: [
      /^\/$/,        // "/" は除外
      /^\/api\/.*/,  // "/api/..." は除外
      /^\/files\/.*/,  // "/files/..." は除外
    ],
  }),
)

function parseBoolLoose(value: unknown): boolean {
  if (typeof value === "boolean") return value
  if (typeof value === "number") return value !== 0
  if (typeof value === "string") {
    const v = value.trim().toLowerCase()
    if (["true", "1", "yes", "y"].includes(v)) return true
    if (["false", "0", "no", "n"].includes(v)) return false
  }
  throw new SyntaxError(`Boolean expected, got ${JSON.stringify(value)}`)
}

self.addEventListener('push', async function (event: any) {
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
    // Kyou系
    url.pathname === '/api/get_kyou' ||
    url.pathname === '/api/get_kmemo' ||
    url.pathname === '/api/get_kc' ||
    url.pathname === '/api/get_urlog' ||
    url.pathname === '/api/get_nlog' ||
    url.pathname === '/api/get_timeis' ||
    url.pathname === '/api/get_mi' ||
    url.pathname === '/api/get_lantana' ||
    url.pathname === '/api/get_rekyou' ||
    url.pathname === '/api/get_git_commit_log' ||
    url.pathname === '/api/get_idf_kyou' ||
    url.pathname === '/api/get_tags_by_id' ||
    url.pathname === '/api/get_texts_by_id' ||
    url.pathname === '/api/get_gkill_notifications_by_id')) {
    event.respondWith(
      (async () => {
        try {
          const reqClone1 = request.clone()
          const reqClone2 = request.clone()

          const body = await reqClone1.json()
          const force_reget = parseBoolLoose(body.force_reget)
          const id = body.target_id ? body.target_id : body.id

          const data_type = new URL(request.url).pathname.replace('/api/get_', '')
          const cacheKey = `/cache/api/${data_type}/${id}`

          const kyou_cache = await caches.open('gkill-post-kyou-cache')
          if (!force_reget) {
            const cached = await kyou_cache.match(cacheKey)
            if (cached) return cached
          }

          const response = await fetch(reqClone2)
          kyou_cache.put(cacheKey, response.clone())
          return response

        } catch (err: any) {
          if ((err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
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
    url.pathname === '/api/get_gkill_info' ||
    url.pathname === '/api/get_all_rep_names' ||
    url.pathname === '/api/get_all_tag_names' ||
    url.pathname === '/api/get_mi_board_list')) {
    event.respondWith(
      (async () => {
        try {
          const reqClone0 = request.clone()
          const reqClone1 = request.clone()

          const body = await reqClone0.json()
          const force_reget = parseBoolLoose(body.force_reget)

          const data_type = new URL(request.url).pathname.replace('/api/get_', '')
          const cacheKey = `/cache/api/${data_type}`

          const config_cache = await caches.open('gkill-post-config-cache')
          if (!force_reget) {
            const cached = await config_cache.match(cacheKey)
            if (cached) return cached
          }

          const response = await fetch(reqClone1)
          config_cache.put(cacheKey, response.clone())
          return response
        } catch (err: any) {
          if ((err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
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

self.addEventListener('fetch', event => {
  const req = event.request;
  if (req.method === 'POST' &&
    new URL(req.url).pathname === '/share-target') {

    event.respondWith((async () => {
      const form = await req.formData();
      const shared_url = form.get('url') as string | null;
      const shared_text = form.get('text') as string | null;
      const shared_title = form.get('title') as string | null;

      const gkill_api = GkillAPI.get_instance()
      const session_id = await gkill_api.get_session_id_from_cookie_store()
      const now = new Date(Date.now())

      const gkill_info_req = new GetGkillInfoRequest()
      gkill_info_req.session_id = session_id
      const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

      let is_saved = false

      if (isUrl(shared_url)) {
        const req = new AddURLogRequest()
        req.session_id = session_id
        req.urlog.url = shared_url
        if (shared_title) {
          req.urlog.title = shared_title
        }
        req.urlog.id = gkill_api.generate_uuid()
        req.urlog.related_time = now
        req.urlog.create_app = "gkill_share"
        req.urlog.create_device = gkill_info_res.device
        req.urlog.create_time = now
        req.urlog.create_user = gkill_info_res.user_id
        req.urlog.update_app = "gkill_share"
        req.urlog.update_device = gkill_info_res.device
        req.urlog.update_time = now
        req.urlog.update_user = gkill_info_res.user_id
        await gkill_api.add_urlog(req)
        is_saved = true
      } else if (isUrl(shared_title)) {
        const req = new AddURLogRequest()
        req.session_id = session_id
        req.urlog.url = shared_title
        req.urlog.id = gkill_api.generate_uuid()
        req.urlog.related_time = now
        req.urlog.create_app = "gkill_share"
        req.urlog.create_device = gkill_info_res.device
        req.urlog.create_time = now
        req.urlog.create_user = gkill_info_res.user_id
        req.urlog.update_app = "gkill_share"
        req.urlog.update_device = gkill_info_res.device
        req.urlog.update_time = now
        req.urlog.update_user = gkill_info_res.user_id
        await gkill_api.add_urlog(req)
        is_saved = true
      } else if (shared_text) {
        const shared_text_lines = String(shared_text).split("http")
        const shared_text_lines_last_line = "http" + shared_text_lines[shared_text.length >= 2 ? shared_text_lines.length - 1 : 0]
        if (isUrl(shared_text)) {
          const req = new AddURLogRequest()
          req.session_id = session_id
          req.urlog.url = shared_text
          if (shared_title) {
            req.urlog.title = shared_title
          }
          req.urlog.id = gkill_api.generate_uuid()
          req.urlog.related_time = now
          req.urlog.create_app = "gkill_share"
          req.urlog.create_device = gkill_info_res.device
          req.urlog.create_time = now
          req.urlog.create_user = gkill_info_res.user_id
          req.urlog.update_app = "gkill_share"
          req.urlog.update_device = gkill_info_res.device
          req.urlog.update_time = now
          req.urlog.update_user = gkill_info_res.user_id
          await gkill_api.add_urlog(req)
          is_saved = true
        } else if (isUrl(shared_text_lines_last_line)) { // AndroidのGoogleアプリだと末尾にURLが入っていることがある
          const req = new AddURLogRequest()
          req.session_id = session_id
          req.urlog.url = shared_text_lines_last_line
          req.urlog.id = gkill_api.generate_uuid()
          req.urlog.related_time = now
          req.urlog.create_app = "gkill_share"
          req.urlog.create_device = gkill_info_res.device
          req.urlog.create_time = now
          req.urlog.create_user = gkill_info_res.user_id
          req.urlog.update_app = "gkill_share"
          req.urlog.update_device = gkill_info_res.device
          req.urlog.update_time = now
          req.urlog.update_user = gkill_info_res.user_id
          await gkill_api.add_urlog(req)
          is_saved = true
        } else {
          const req = new AddKmemoRequest()
          req.session_id = session_id
          req.kmemo.content = shared_text
          req.kmemo.id = gkill_api.generate_uuid()
          req.kmemo.related_time = now
          req.kmemo.create_app = "gkill_share"
          req.kmemo.create_device = gkill_info_res.device
          req.kmemo.create_time = now
          req.kmemo.create_user = gkill_info_res.user_id
          req.kmemo.update_app = "gkill_share"
          req.kmemo.update_device = gkill_info_res.device
          req.kmemo.update_time = now
          req.kmemo.update_user = gkill_info_res.user_id
          await gkill_api.add_kmemo(req)
          is_saved = true
        }
      }

      return Response.redirect('/saihate?is_saved=' + is_saved, 303);
    })());
  }
});