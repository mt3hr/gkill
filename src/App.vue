<script setup lang="ts">
import { type Ref, ref } from 'vue';
import { RouterView } from 'vue-router'
import { VLocaleProvider } from 'vuetify/components';
import { useTheme } from 'vuetify'
import { GkillAPI } from './classes/api/gkill-api';

const theme = useTheme()
const use_dark_theme = GkillAPI.get_gkill_api().get_use_dark_theme()
if (use_dark_theme) {
  theme.global.name.value = 'gkill_dark_theme'
} else {
  theme.global.name.value = 'gkill_theme'
}

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