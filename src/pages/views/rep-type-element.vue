<template>
    <ApplicationConfigStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :folder_name="folder_name" :is_open="is_open" :struct_obj="struct_obj" />
</template>
<script setup lang="ts">
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue'
import type { RepTypeElementEmits } from './rep-type-element-emits'
import type { RepTypeElementProps } from './rep-type-element-props'

defineProps<RepTypeElementProps>()
defineEmits<RepTypeElementEmits>()
</script>
