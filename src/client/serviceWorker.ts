/// <reference lib="webworker" />
import delete_gkill_kyou_cache from './classes/delete-gkill-cache';
import { isUrl } from './classes/looks-like-url';
import { clientsClaim } from 'workbox-core'
import { cleanupOutdatedCaches, precacheAndRoute, createHandlerBoundToURL, } from 'workbox-precaching'
import { registerRoute, NavigationRoute } from 'workbox-routing'
import { shouldCacheResponse, parseBoolLoose } from './classes/service-worker-utils';

export default null

self.skipWaiting()
clientsClaim()
declare let clients: Clients;
declare let self: ServiceWorkerGlobalScope

cleanupOutdatedCaches()

precacheAndRoute(self.__WB_MANIFEST, {
  directoryIndex: null as unknown as string,
})

// SPA の app-shell (index.html) フォールバック。ただし / と /api/ は除外
registerRoute(
  new NavigationRoute(createHandlerBoundToURL('index.html'), {
    denylist: [
      /^\/$/,        // "/" は除外
      /^\/api\/.*/,  // "/api/..." は除外
      /^\/files\/.*/,  // "/files/..." は除外
      /^\/zip_cache\/.*/,  // "/zip_cache/..." は除外（ZIP展開キャッシュ）
      /^\/resources\/manual\/.*/,  // "/resources/manual/..." は除外（ヘルプHTML）
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
          if (await shouldCacheResponse(response, true)) {
            kyou_cache.put(cacheKey, response.clone())
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
          const reqClone1 = request.clone()
          const reqClone2 = request.clone()

          const body = await reqClone1.json()
          const force_reget = parseBoolLoose(body.force_reget)
          const id = body.kyou_id

          const cacheKey = `/cache/api/plugin_content_html/${id}`

          const kyou_cache = await caches.open('gkill-post-kyou-cache')
          if (!force_reget) {
            const cached = await kyou_cache.match(cacheKey)
            if (cached) return cached
          }

          const response = await fetch(reqClone2)
          if (await shouldCacheResponse(response, true)) {
            kyou_cache.put(cacheKey, response.clone())
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
          if (await shouldCacheResponse(response, false)) {
            config_cache.put(cacheKey, response.clone())
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

self.addEventListener('fetch', event => {
  const req = event.request;
  if (req.method === 'POST' &&
    new URL(req.url).pathname === '/share-target') {

    event.respondWith((async () => {
      let is_saved = false
      try {
        const form = await req.formData();
        const shared_url = form.get('url') as string | null;
        const shared_text = form.get('text') as string | null;
        const shared_title = form.get('title') as string | null;

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

        const add_urlog = async (url: string, title: string): Promise<void> => {
          await fetch('/api/add_urlog', {
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
                url,
                title,
                description: "",
                favicon_image: "",
                thumbnail_image: "",
              },
            }),
          })
        }

        const add_kmemo = async (content: string): Promise<void> => {
          await fetch('/api/add_kmemo', {
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
                content,
              },
            }),
          })
        }

        if (isUrl(shared_url)) {
          await add_urlog(shared_url, shared_title ?? "")
          is_saved = true
        } else if (isUrl(shared_title)) {
          await add_urlog(shared_title, "")
          is_saved = true
        } else if (shared_text) {
          const shared_text_lines = String(shared_text).split("http")
          const shared_text_lines_last_line = "http" + shared_text_lines[shared_text.length >= 2 ? shared_text_lines.length - 1 : 0]
          if (isUrl(shared_text)) {
            await add_urlog(shared_text, shared_title ?? "")
            is_saved = true
          } else if (isUrl(shared_text_lines_last_line)) { // AndroidのGoogleアプリだと末尾にURLが入っていることがある
            await add_urlog(shared_text_lines_last_line, "")
            is_saved = true
          } else {
            await add_kmemo(shared_text)
            is_saved = true
          }
        }
      } catch (e) {
        console.error('[SW] share-target error:', e)
        is_saved = false
      }
      return Response.redirect('/saihate?is_saved=' + is_saved, 303);
    })());
  }
});
