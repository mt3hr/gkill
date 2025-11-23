<template>
    <v-btn dark color="primary" @click="emits('request_open_manage_share_kyou_dialog')">{{
        i18n.global.t('MANAGE_SHARE_KYOU_TITLE') }}</v-btn>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ManageShareButtonEmits } from './manage-share-button-emits'

const emits = defineEmits<ManageShareButtonEmits>()
</script>
