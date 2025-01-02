<script setup lang="ts">
import { nextTick, type Ref, ref } from 'vue';
import { RouterView } from 'vue-router'
import { VLocaleProvider } from 'vuetify/components';
import { GkillAPI } from './classes/api/gkill-api';
import type { GkillError } from './classes/api/gkill-error';
import type { GkillMessage } from './classes/api/gkill-message';
import { GetGkillNotificationPublicKeyRequest } from './classes/api/req_res/get-gkill-notification-public-key-request';
import { RegisterGkillNotificationRequest } from './classes/api/req_res/register-gkill-notification-request';
const locale: Ref<string> = ref(window.navigator.language)

// プッシュ通知登録用
async function subscribe(vapidPublicKey: string) {
  if (!vapidPublicKey || vapidPublicKey === "") {
    return
  }
  await navigator.serviceWorker.ready
    .then(function (registration) {
      vapidPublicKey = vapidPublicKey;
      return registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
      });
    })
    .then(async function (subscription) {
      const req = new RegisterGkillNotificationRequest()
      req.session_id = GkillAPI.get_gkill_api().get_session_id()
      req.subscription = JSON.stringify(subscription)
      req.public_key = vapidPublicKey
      const res = await GkillAPI.get_gkill_api().register_gkill_notification(req)
      if (res.errors && res.errors.length !== 0) {
        write_errors(res.errors)
        return
      }
      if (res.messages && res.messages.length !== 0) {
        write_messages(res.messages)
      }
    })
    .catch(err => console.error(err));
}
// プッシュ通知登録用
function urlBase64ToUint8Array(base64String: string) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/\-/g, '+')
    .replace(/_/g, '/');
  const rawData = window.atob(base64);
  return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
}

nextTick(() => register_mi_task_notification())

// プッシュ通知登録用
async function register_mi_task_notification(): Promise<void> {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('gkill_mi_task_notification');
    await navigator.serviceWorker.ready
      .then(function (registration) {
        return registration.pushManager.getSubscription();
      })
      .then(async function (subscription) {
        if (!subscription) {
          const req = new GetGkillNotificationPublicKeyRequest()
          req.session_id = GkillAPI.get_gkill_api().get_session_id()
          const res = await GkillAPI.get_gkill_api().get_gkill_notification_public_key(req)
          if (res.errors && res.errors.length !== 0) {
            write_errors(res.errors)
            return
          }
          if (res.messages && res.messages.length !== 0) {
            write_messages(res.messages)
          }
          subscribe(res.gkill_notification_public_key)
        } else {
          console.log(
            JSON.stringify({
              subscription: subscription,
            })
          )
        }
      })
  }
}

const messages: Ref<Array<{ message: string, id: string, show_snackbar: boolean }>> = ref([])

async function write_errors(errors: Array<GkillError>) {
  const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
  for (let i = 0; i < errors.length; i++) {
    if (errors[i] && errors[i].error_message) {
      received_messages.push({
        message: errors[i].error_message,
        id: GkillAPI.get_instance().generate_uuid(),
        show_snackbar: true,
      })
    }
  }
  messages.value.push(...received_messages)
  sleep(2500).then(() => {
    for (let i = 0; i < received_messages.length; i++) {
      messages.value.splice(0, 1)
    }
  })
}

async function write_messages(messages_: Array<GkillMessage>) {
  const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
  for (let i = 0; i < messages_.length; i++) {
    if (messages_[i] && messages_[i].message) {
      received_messages.push({
        message: messages_[i].message,
        id: GkillAPI.get_instance().generate_uuid(),
        show_snackbar: true,
      })
    }
  }
  messages.value.push(...received_messages)
  sleep(2500).then(() => {
    for (let i = 0; i < received_messages.length; i++) {
      messages.value.splice(0, 1)
    }
  })
}
const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

</script>

<template>
  <table>
    <tr>
      <td>
        <div id="control-height"></div>
      </td>
      <td>
        <v-app>
          <VLocaleProvider :locale="locale">
            <RouterView />
          </VLocaleProvider>
        </v-app>
        <div class="alert_container">
          <v-slide-y-transition group>
            <v-alert v-for="message in messages" theme="dark">
              {{ message.message }}
            </v-alert>
          </v-slide-y-transition>
        </div>
      </td>
    </tr>
  </table>
</template>

<style scoped></style>
<style>
#control-height {
  height: 100vh;
  width: 0;
  position: absolute;
  overflow-y: hidden;
}

.alert_container {
  position: fixed;
  top: 60px;
  right: 10px;
  display: grid;
  grid-gap: .5em;
  z-index: 100000000;
}
</style>