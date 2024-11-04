<script setup lang="ts">
//TODO requested_update_last_added_tag
//TODO requested_update_last_added_tagをトップまでに持ってくる。router超えて使えるようにしたい。Cookie？
import { type Ref, ref } from 'vue';
import { RouterView } from 'vue-router'
import { VLocaleProvider } from 'vuetify/components';
const locale: Ref<string> = ref(window.navigator.language)
</script>

<template>
  <v-app>
    <VLocaleProvider :locale="locale">
      <RouterView />
    </VLocaleProvider>
  </v-app>
  <div id="control-height"></div>
</template>

<style scoped></style>
<style>
#control-height {
  height: 100vh;
  width: 0;
  position: absolute;
}
* {
  scrollbar-width: none;
}
</style>