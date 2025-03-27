<template>
    <v-btn dark color="primary" @click="emits('request_open_share_mi_dialog')">共有</v-btn>
</template>
<script lang="ts" setup>
import type { ShareButtonEmits } from './share-button-emits'

const emits = defineEmits<ShareButtonEmits>()
</script>
