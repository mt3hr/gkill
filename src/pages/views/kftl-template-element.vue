<template>
    <ApplicationConfigStructContextMenu :struct_obj="struct_obj" :application_config="application_config"
        :folder_name="''" :gkill_api="gkill_api" :is_open="true" />
</template>
<script setup lang="ts">
import type { KFTLTemplateElementEmits } from './kftl-template-element-emits'
import type { KFTLTemplateElementProps } from './kftl-template-element-props'

defineProps<KFTLTemplateElementProps>()
defineEmits<KFTLTemplateElementEmits>()
</script>
