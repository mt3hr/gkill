<script setup lang="ts">
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