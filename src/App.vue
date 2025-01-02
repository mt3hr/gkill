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
</style>