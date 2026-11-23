<script setup lang="ts">
import '@/styles/gkill-global.css'
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue';
import { RouterView } from 'vue-router'
import { VLocaleProvider } from 'vuetify/components';
import { useTheme } from 'vuetify'
import { GkillAPI } from './classes/api/gkill-api';
import SaihateStarsOverlay from './pages/views/saihate-stars-overlay.vue'
import SnowFallOverlay from './pages/views/snow-fall-overlay.vue';

// ロケール適用 (メッセージの動的ロードを含む) は main.ts が mount 前に済ませている

const theme = useTheme()
const use_dark_theme = GkillAPI.get_gkill_api().get_use_dark_theme()
if (use_dark_theme) {
  theme.global.name.value = 'gkill_dark_theme'
} else {
  theme.global.name.value = 'gkill_theme'
}

const locale: Ref<string> = ref(i18n.global.locale)


</script>


<template>
  <div>
    <SaihateStarsOverlay v-if="theme.global.name.value === 'gkill_dark_theme'" />
    <SnowFallOverlay v-if="theme.global.name.value === 'gkill_theme'" />
    <table>
        <tbody>
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
        </tbody>
    </table>
  </div>
</template>
