<template>
    <ShareButton />
    <ManageShareButton />
</template>
<script setup lang="ts">
import ManageShareButton from './manage-share-button.vue'
import ShareButton from './share-button.vue'
import type { miShareFooterEmits } from './mi-share-footer-emits'
import type { miShareFooterProps } from './mi-share-footer-props'

const props = defineProps<miShareFooterProps>()
const emits = defineEmits<miShareFooterEmits>()
</script>
