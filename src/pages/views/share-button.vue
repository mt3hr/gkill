<template>
    <v-btn dark color="primary" @click="emits('request_open_share_mi_dialog')">{{ $t("SHARE_TITLE") }}</v-btn>
</template>
<script lang="ts" setup>
import type { ShareButtonEmits } from './share-button-emits'
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

const emits = defineEmits<ShareButtonEmits>()
</script>
