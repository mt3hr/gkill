<template>
    <v-btn @click="emits('request_open_manage_share_mi_dialog')">共有管理</v-btn>
</template>
<script lang="ts" setup>
import type { ManageShareButtonEmits } from './manage-share-button-emits'

const emits = defineEmits<ManageShareButtonEmits>()
</script>
