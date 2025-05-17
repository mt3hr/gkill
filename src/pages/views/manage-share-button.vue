<template>
    <v-btn dark color="primary" @click="emits('request_open_manage_share_kyou_dialog')">共有管理</v-btn>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ManageShareButtonEmits } from './manage-share-button-emits'

const emits = defineEmits<ManageShareButtonEmits>()
</script>
