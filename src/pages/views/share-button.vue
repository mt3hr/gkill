<template>
    <v-btn dark color="primary" @click="emits('request_open_share_kyou_dialog')">{{ $t("SHARE_TITLE") }}</v-btn>
</template>
<script lang="ts" setup>
import type { ShareButtonEmits } from './share-button-emits'

const emits = defineEmits<ShareButtonEmits>()
</script>
