<script setup lang="ts">
//TODO requested_update_last_added_tag
//TODO requested_update_last_added_tagをトップまでに持ってくる。router超えて使えるようにしたい。Cookie？
import { RouterView } from 'vue-router'
</script>

<template>
  <div id="control-height"></div>
  <RouterView />
</template>

<style scoped></style>
<style>
#control-height {
  height: 100vh;
  width: 0;
  position: absolute;
}
</style>