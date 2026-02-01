<template>
    <div></div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import router from '@/router';
import { useRoute } from 'vue-router';

// redirect
router.replace("/shared_page?share_id=" + useRoute().query.share_id!.toString())
</script>