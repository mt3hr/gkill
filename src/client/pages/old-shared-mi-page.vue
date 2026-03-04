<template>
    <div></div>
</template>

<script lang="ts" setup>
import router from '@/router';
import { useRoute } from 'vue-router';
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'

// redirect
await resetDialogHistory()
router.replace("/shared_page?share_id=" + useRoute().query.share_id!.toString())
</script>