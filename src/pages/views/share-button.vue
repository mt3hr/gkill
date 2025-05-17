<template>
    <v-btn dark color="primary" @click="emits('request_open_share_kyou_dialog')">{{ i18n.global.t("SHARE_TITLE") }}</v-btn>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ShareButtonEmits } from './share-button-emits'

const emits = defineEmits<ShareButtonEmits>()
</script>
