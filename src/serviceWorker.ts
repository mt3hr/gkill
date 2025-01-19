/// <reference lib="webworker" />
import { precacheAndRoute } from 'workbox-precaching'
export default null
declare let self: ServiceWorkerGlobalScope
declare var clients: Clients;

precacheAndRoute(self.__WB_MANIFEST)

self.addEventListener('push', function (event: any) {
  const title = 'gkill'
  const data = event.data.json()
  const options = {
    body: data.content,
    requireInteraction: true,
    data: data,
    timestamp: Math.floor(data.time),
  }
  event.waitUntil(self.registration.showNotification(title, options))
})

self.addEventListener('notificationclick', function (event) {
  const data = event.notification.data
  event.notification.close()
  event.waitUntil(
    clients.openWindow(data.url)
  )
})