self.addEventListener('push', event => {
  const title = 'gkill'
  const data = event.data.json()
  const options = {
    body: data.content,
    requireInteraction: true,
    data: data,
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