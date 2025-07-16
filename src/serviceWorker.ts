/// <reference lib="webworker" />
import { precacheAndRoute } from 'workbox-precaching'
import { clientsClaim } from 'workbox-core'
import delete_gkill_kyou_cache from './classes/delete-gkill-cache';
export default null

self.skipWaiting()
clientsClaim()
declare let self: ServiceWorkerGlobalScope
declare let clients: Clients;

precacheAndRoute(self.__WB_MANIFEST)

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