/// <reference lib="webworker" />
import { precacheAndRoute } from 'workbox-precaching'
import delete_gkill_kyou_cache from './classes/delete-gkill-cache';
export default null
declare let self: ServiceWorkerGlobalScope
declare let clients: Clients;

precacheAndRoute(self.__WB_MANIFEST)

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
        const reqClone1 = request.clone()
        const reqClone2 = request.clone()

        const body = await reqClone1.json()
        const force_reget = body.force_reget
        let id = body.id
        if (!id) {
          id = body.target_id
        }
        if (!id) {
          return fetch(reqClone2)
        }

        const data_type = new URL(request.url).pathname.replace('/api/get_', '')
        const cacheKey = `/cache/api/${data_type}/${id}`

        const cache = await caches.open('gkill-post-kyou-cache')
        if (JSON.parse(force_reget).toString().toLowerCase() !== "true") {
          const cached = await cache.match(cacheKey)
          if (cached) return cached
        }

        const response = await fetch(reqClone2)
        cache.put(cacheKey, response.clone())
        return response
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
        const reqClone0 = request.clone()
        const reqClone1 = request.clone()

        const body = await reqClone0.json()
        const force_reget = body.force_reget

        const data_type = new URL(request.url).pathname.replace('/api/get_', '')
        const cacheKey = `/cache/api/${data_type}`

        const cache = await caches.open('gkill-post-config-cache')
        if (JSON.parse(force_reget).toString().toLowerCase() !== "true") {
          const cached = await cache.match(cacheKey)
          if (cached) return cached
        }

        const response = await fetch(reqClone1)
        cache.put(cacheKey, response.clone())
        return response
      })()
    )
  }
})