<script setup lang="ts">
//TODO requested_update_last_added_tag
//TODO requested_update_last_added_tagをトップまでに持ってくる。router超えて使えるようにしたい。Cookie？
import { RouterView } from 'vue-router'
</script>

<template>
  <v-app>
    <RouterView />
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
</style>